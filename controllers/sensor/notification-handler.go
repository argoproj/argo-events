package sensor

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sv1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sv1 "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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
func (se *sensorExecutor) resyncSensor(ctx context.Context) (cache.Controller, error) {
	se.log.Info().Msg("watching sensor updates")
	source := se.newSensorWatch()
	_, controller := cache.NewInformer(
		source,
		&v1alpha1.Sensor{},
		0,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				if newSensor, ok := new.(*v1alpha1.Sensor); ok {
					se.log.Info().Msg("detected Sensor update.")
					se.sensor = newSensor
					if se.sensor.Status.Phase == v1alpha1.NodePhaseError {
						se.log.Error().Msg("sensor is in error phase, stopping the server...")
						// Shutdown server as sensor should not process any event in error state
						se.server.Shutdown(context.Background())
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (se *sensorExecutor) newSensorWatch() *cache.ListWatch {
	x := se.sensorClient.RESTClient()
	resource := "sensors"
	name := se.sensor.Name
	namespace := se.sensor.Namespace
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// WatchNotifications watches and handles signals sent by the gateway the sensor is interested in.
func (se *sensorExecutor) WatchSignalNotifications() {
	// watch sensor updates
	go se.resyncSensor(context.Background())
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
	defer se.persistUpdates()
	body, err := ioutil.ReadAll(r.Body)
	gatewaySignal := sv1alpha.Event{}
	err = json.Unmarshal(body, &gatewaySignal)
	if err != nil {
		se.log.Error().Err(err).Msg("failed to decode notification")
		common.SendErrorResponse(w)
		return
	}

	// validate the signal is from gateway of interest
	var isSignalValid bool
	for _, signal := range se.sensor.Spec.Signals {
		if signal.Name == gatewaySignal.Context.Source.Host {
			isSignalValid = true
			break
		}
	}

	if isSignalValid {
		// send success response back to gateway as it is a valid notification
		common.SendSuccessResponse(w)

		// if signal is already completed, don't process the new event.
		if getNodeByName(se.sensor, gatewaySignal.Context.Source.Host).Phase == v1alpha1.NodePhaseComplete {
			se.log.Warn().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("signal is already completed")
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

		// to trigger the sensor action/s we need to check if all signals are completed and sensor is active
		completed := se.sensor.AreAllNodesSuccess(v1alpha1.NodeTypeSignal) && se.sensor.Status.Phase == v1alpha1.NodePhaseActive
		if completed {
			se.log.Info().Msg("all signals are marked completed")

			// trigger action/s
			for _, trigger := range se.sensor.Spec.Triggers {
				se.markNodePhase(trigger.Name, v1alpha1.NodePhaseActive, "active")
				// execute trigger action
				err := se.executeTrigger(trigger)
				if err != nil {
					se.log.Error().Str("trigger-name", trigger.Name).Err(err).Msg("trigger failed to execute")
					se.markNodePhase(trigger.Name, v1alpha1.NodePhaseError, err.Error())
					se.sensor.Status.Phase = v1alpha1.NodePhaseError
					return
				}
				// mark trigger as complete.
				se.markNodePhase(trigger.Name, v1alpha1.NodePhaseComplete, "trigger completed successfully")
			}

			if !se.sensor.Spec.Repeat {
				// todo: unsubscribe from pub-sub system
				se.server.Shutdown(context.Background())
			}
		}
	} else {
		se.log.Error().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("unknown signal source")
		common.SendErrorResponse(w)
		return
	}

}

// persist the updates to the Sensor resource
func (se *sensorExecutor) persistUpdates() {
	var err error
	sensorClient := se.sensorClient.Sensors(se.sensor.ObjectMeta.Namespace)
	// no need to reassign the sensor object here as resyncSensor method will take care of that
	_, err = sensorClient.Update(se.sensor)
	if err != nil {
		se.log.Warn().Err(err).Msg("error updating sensor")
		if errors.IsConflict(err) {
			return
		}
		se.log.Info().Msg("re-applying updates on latest version and retrying update")
		err = se.reapplyUpdate()
		if err != nil {
			se.log.Error().Err(err).Msg("failed to re-apply update")
			return
		}
	}
	se.log.Info().Msg("sensor updated successfully")
	time.Sleep(1 * time.Second)
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
