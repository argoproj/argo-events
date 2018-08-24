package sensor_controller

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sv1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sv1 "github.com/argoproj/argo-events/pkg/sensor-client/clientset/versioned/typed/sensor/v1alpha1"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"sync"
	"time"
)

// sensorExecutor contains execution context for sensor operation
type sensorExecutor struct {
	// sensorClient is the client for sensor
	sensorClient sv1.ArgoprojV1alpha1Interface
	// kubeClient is the kubernetes client
	kubeClient kubernetes.Interface
	// config is the cluster config
	config *rest.Config
	// sensor object
	sensor *v1alpha1.Sensor
	// http server which exposes the sensor to gateway/s
	server *http.Server
	// logger for the sensor
	log zerolog.Logger
	// waitgroup
	wg *sync.WaitGroup
}

func NewSensorExecutor(sensorClient sv1.ArgoprojV1alpha1Interface, kubeClient kubernetes.Interface, config *rest.Config, sensor *v1alpha1.Sensor, log zerolog.Logger, wg *sync.WaitGroup) *sensorExecutor {
	return &sensorExecutor{
		sensorClient: sensorClient,
		kubeClient:   kubeClient,
		config:       config,
		sensor:       sensor,
		log:          log,
		wg:           wg,
	}
}

// resyncs the sensor object for status updates
func (se *sensorExecutor) watchSensorUpdates() {
	watcher, err := se.sensorClient.Sensors(se.sensor.Namespace).Watch(metav1.ListOptions{})
	if err != nil {
		se.log.Panic().Err(err).Msg("failed to get sensor updates")
	}
	for update := range watcher.ResultChan() {
		se.sensor = update.Object.(*v1alpha1.Sensor).DeepCopy()
		se.log.Info().Msg("sensor state updated")
		if se.sensor.Status.Phase == v1alpha1.NodePhaseError {
			se.log.Error().Msg("sensor is in error phase, stopping the server...")
			// Shutdown server as sensor should not process any event in error state
			se.server.Shutdown(context.Background())
		}
	}
}

// WatchNotifications watches and handles signals sent by the gateway the sensor is interested in.
func (se *sensorExecutor) WatchSignalNotifications() {
	// watch sensor updates
	go se.watchSensorUpdates()
	srv := &http.Server{Addr: fmt.Sprintf(":%d", common.SensorServicePort)}
	http.HandleFunc("/", se.handleSignals)
	go func() {
		se.log.Info().Str("port", string(common.SensorServicePort)).Msg("sensor started listening")
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			se.log.Warn().Err(err).Msg("sensor server stopped")
			se.wg.Done()
		}
	}()
	se.server = srv
}

// Handles signals received from gateway/s
func (se *sensorExecutor) handleSignals(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	gatewaySignal := sv1alpha.Event{}
	err = json.Unmarshal(body, &gatewaySignal)
	if err != nil {
		se.log.Error().Err(err).Msg("failed to decode notification")
		common.SendErrorResponse(&w)
		return
	}

	// validate the signal is from gateway of interest
	var validSignal bool
	for _, signal := range se.sensor.Spec.Signals {
		if signal.Name == gatewaySignal.Context.Source.Host {
			validSignal = true
			break
		}
	}

	if validSignal{
		// if signal is already completed, don't process the new event.
		// Todo: what should be done when rate of signals received exceeds the complete sensor processing time?
		// maybe add queueing logic
		if getNodeByName(se.sensor, gatewaySignal.Context.Source.Host).Phase == v1alpha1.NodePhaseComplete {
			se.log.Warn().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("signal is already completed")
			common.SendErrorResponse(&w)
			return
		}

		// process the signal
		se.log.Info().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("processing the signal")
		node := getNodeByName(se.sensor, gatewaySignal.Context.Source.Host)
		node.LatestEvent = &v1alpha1.EventWrapper{
			Event: gatewaySignal,
			Seen:  true,
		}
		se.markNodePhase(node.Name, v1alpha1.NodePhaseComplete, "signal is completed")

		se.sensor, err = se.updateSensor()
		if err != nil {
			err = se.reapplyUpdate()
			if err != nil {
				se.log.Error().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("failed to update signal node state")
				common.SendErrorResponse(&w)
				return
			}
		}

		// to trigger the sensor action/s we first need to check if all signals are completed and sensor is active
		completed := se.sensor.AreAllNodesSuccess(v1alpha1.NodeTypeSignal) && se.sensor.Status.Phase == v1alpha1.NodePhaseActive
		if completed {
			se.log.Info().Msg("all signals are processed")

			// trigger action/s
			for _, trigger := range se.sensor.Spec.Triggers {
				se.markNodePhase(trigger.Name, v1alpha1.NodePhaseActive, "trigger is about to run")
				// update the sensor for marking trigger as active
				se.sensor, err = se.updateSensor()
				err := se.executeTrigger(trigger)
				if err != nil {
					// Todo: should we let other triggers to run?
					se.log.Error().Str("trigger-name", trigger.Name).Err(err).Msg("trigger failed to execute")
					se.markNodePhase(trigger.Name, v1alpha1.NodePhaseError, err.Error())
					se.sensor.Status.Phase = v1alpha1.NodePhaseError
					// update the sensor object with error state
					se.sensor, err = se.updateSensor()
					if err != nil {
						err = se.reapplyUpdate()
						if err != nil {
							se.log.Error().Err(err).Msg("failed to update sensor phase")
							common.SendErrorResponse(&w)
							return
						}
					}
				}
				// mark trigger as complete.
				se.markNodePhase(trigger.Name, v1alpha1.NodePhaseComplete, "trigger completed successfully")
				se.sensor, err = se.updateSensor()
				if err != nil {
					err = se.reapplyUpdate()
					if err != nil {
						se.log.Error().Err(err).Msg("failed to update trigger completion state")
						common.SendErrorResponse(&w)
						return
					}
				}
			}

			if !se.sensor.Spec.Repeat {
				common.SendSuccessResponse(&w)
				// todo: unsubscribe from pub-sub system
				se.server.Shutdown(context.Background())
			}
		}
	} else {
		se.log.Error().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("unknown signal source")
		common.SendErrorResponse(&w)
		return
	}

}

// update sensor resource
func (se *sensorExecutor) updateSensor() (*v1alpha1.Sensor, error) {
	return se.sensorClient.Sensors(se.sensor.Namespace).Update(se.sensor)
}

// reapply the update to sensor
func (se *sensorExecutor) reapplyUpdate() error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		sClient := se.sensorClient.Sensors(se.sensor.Namespace)
		s, err := sClient.Get(se.sensor.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		s.Status = se.sensor.Status
		se.sensor, err = sClient.Update(s)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
}

// mark the node with a phase, retuns the node
func (se *sensorExecutor) markNodePhase(nodeName string, phase v1alpha1.NodePhase, message ...string) *v1alpha1.NodeStatus {
	node := getNodeByName(se.sensor, nodeName)
	if node == nil {
		se.log.Panic().Str("node-name", nodeName).Msg("node is uninitialized")
	}
	if node.Phase != phase {
		se.log.Info().Str("type", string(node.Type)).Str("node-name", node.Name).Str("phase", string(node.Phase))
		node.Phase = phase
	}
	if len(message) > 0 && node.Message != message[0] {
		se.log.Info().Str("type", string(node.Type)).Str("node-name", node.Name).Str("phase", string(node.Phase)).Str("message", message[0])
		node.Message = message[0]
	}
	if node.IsComplete() && node.CompletedAt.IsZero() {
		node.CompletedAt = metav1.Time{Time: time.Now().UTC()}
		se.log.Info().Str("type", string(node.Type)).Str("node-name", node.Name).Msg("completed")
	}
	se.sensor.Status.Nodes[node.ID] = *node
	return node
}
