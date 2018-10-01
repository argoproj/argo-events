package sensor

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	ss_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"sync"
	"time"
)

// sensorExecutionCtx contains execution context for sensor
type sensorExecutionCtx struct {
	// sensorClient is the client for sensor
	sensorClient clientset.Interface
	// kubeClient is the kubernetes client
	kubeClient kubernetes.Interface
	// ClientPool manages a pool of dynamic clients.
	clientPool dynamic.ClientPool
	// DiscoveryClient implements the functions that discover server-supported API groups,
	// versions and resources.
	discoveryClient discovery.DiscoveryInterface
	// sensor object
	sensor *v1alpha1.Sensor
	// http server which exposes the sensor to gateway/s
	server *http.Server
	// logger for the sensor
	log zerolog.Logger
	// waitgroup
	wg *sync.WaitGroup
	// queue is internal queue to manage incoming events
	queue chan *sensorEventWrapper
}

// sensorEventWrapper is a wrapper around event received from gateway and the signal which represents the event
type sensorEventWrapper struct {
	event  *ss_v1alpha1.Event
	signal *v1alpha1.Signal
	writer http.ResponseWriter
}

// NewsensorExecutionCtx returns a new sensor execution context.
func NewsensorExecutionCtx(sensorClient clientset.Interface, kubeClient kubernetes.Interface,
	clientPool dynamic.ClientPool, discoveryClient discovery.DiscoveryInterface,
	sensor *v1alpha1.Sensor, log zerolog.Logger, wg *sync.WaitGroup) *sensorExecutionCtx {
	return &sensorExecutionCtx{
		sensorClient:    sensorClient,
		kubeClient:      kubeClient,
		clientPool:      clientPool,
		discoveryClient: discoveryClient,
		sensor:          sensor,
		log:             log,
		wg:              wg,
		queue:           make(chan *sensorEventWrapper),
	}
}

// resyncs the sensor object for status updates
func (se *sensorExecutionCtx) resyncSensor(ctx context.Context) (cache.Controller, error) {
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

func (se *sensorExecutionCtx) newSensorWatch() *cache.ListWatch {
	x := se.sensorClient.ArgoprojV1alpha1().RESTClient()
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

// processTrigger checks if all signal nodes are complete and if all signal nodes are complete then it starts executing triggers
func (se *sensorExecutionCtx) processTrigger() error {
	// to trigger the sensor action/s we need to check if all signals are completed and sensor is active
	completed := se.sensor.AreAllNodesSuccess(v1alpha1.NodeTypeSignal) && se.sensor.Status.Phase == v1alpha1.NodePhaseActive
	if completed {
		se.log.Info().Msg("all signals are marked completed")

		// trigger action/s
		for _, trigger := range se.sensor.Spec.Triggers {
			// execute trigger action
			err := se.executeTrigger(trigger)
			if err != nil {
				se.log.Error().Str("trigger-name", trigger.Name).Err(err).Msg("trigger failed to execute")
				se.markNodePhase(trigger.Name, v1alpha1.NodePhaseError, err.Error())
				se.sensor.Status.Phase = v1alpha1.NodePhaseError
				return err
			}
			// mark trigger as complete.
			se.markNodePhase(trigger.Name, v1alpha1.NodePhaseComplete, "trigger completed successfully")
		}
		if !se.sensor.Spec.Repeat {
			se.server.Shutdown(context.Background())
		}
	}
	return nil
}

// processSignal processes signal received by sensor, validates it, updates the state of the node representing the signal
// and executes trigger if all node with nodetype signal are complete.
func (se *sensorExecutionCtx) processSignal(gwEventWrapper *sensorEventWrapper) {
	defer se.persistUpdates()

	// check if sensor is in error state
	if se.sensor.Status.Phase == v1alpha1.NodePhaseError {
		se.log.Warn().Str("signal-src", gwEventWrapper.event.Context.Source.Host).Msg("sensor is in error state. can't process the signal")
		common.SendErrorResponse(gwEventWrapper.writer)
		return
	}
	se.log.Info().Str("signal-src", gwEventWrapper.event.Context.Source.Host).Msg("signal source")

	// apply filters if any.
	// error is thrown if some problem occurs during filtering the signal
	// apply filters if any
	ok, err := se.filterEvent(gwEventWrapper.signal.Filters, gwEventWrapper.event)
	if err != nil {
		se.log.Error().Err(err).Str("signal-name", gwEventWrapper.event.Context.Source.Host).Err(err).Msg("failed to apply filter")
		common.SendErrorResponse(gwEventWrapper.writer)
		return
	}
	if !ok {
		se.log.Error().Str("signal-name", gwEventWrapper.event.Context.Source.Host).Err(err).Msg("failed to apply filter")
		common.SendErrorResponse(gwEventWrapper.writer)
		return
	}

	if ok {
		// send success response back to gateway as it is a valid notification
		common.SendSuccessResponse(gwEventWrapper.writer)
		se.log.Info().Str("signal-name", gwEventWrapper.event.Context.Source.Host).Msg("processing the signal")
		node := getNodeByName(se.sensor, gwEventWrapper.event.Context.Source.Host)
		// if signal is already completed, don't process the new event. This will still mark node as complete.
		if node.Phase == v1alpha1.NodePhaseComplete {
			se.log.Warn().Str("signal-name", gwEventWrapper.event.Context.Source.Host).Msg("signal is already completed")
			return
		}
		// mark this signal/event as seen. this event will be set in sensor node.
		node.LatestEvent = &v1alpha1.EventWrapper{
			Event: *gwEventWrapper.event,
			Seen:  true,
		}
		se.sensor.Status.Nodes[node.ID] = *node
		se.markNodePhase(node.Name, v1alpha1.NodePhaseComplete, "signal is completed")
		err := se.processTrigger()
		if err != nil {
			se.log.Error().Err(err).Str("signal-name", gwEventWrapper.event.Context.Source.Host).Msg("failed to execute triggers")
		}
		return
	} else {
		se.log.Error().Str("signal-name", gwEventWrapper.event.Context.Source.Host).Msg("unknown signal source")
		common.SendErrorResponse(gwEventWrapper.writer)
		return
	}
}

// processQueue processes the event sent from gateway to this sensor.
func (se *sensorExecutionCtx) processQueue() {
	for {
		gwEventWrapper := <-se.queue
		se.processSignal(gwEventWrapper)
	}
}

// WatchNotifications watches and handles signals sent by the gateway the sensor is interested in.
func (se *sensorExecutionCtx) WatchSignalNotifications() {
	go se.processQueue()
	// watch sensor updates and resyncs the local sensor object.
	go se.resyncSensor(context.Background())

	// create a http server. this server listens for notifications/signals/events from gateway.
	srv := &http.Server{Addr: fmt.Sprintf(":%s", common.SensorServicePort)}

	// add a handler to handle incoming events
	http.HandleFunc("/", se.handleSignals)

	go func() {
		se.log.Info().Str("port", string(common.SensorServicePort)).Msg("sensor started listening")
		if err := srv.ListenAndServe(); err != nil {
			// exit main process
			se.wg.Done()
			if err == http.ErrServerClosed {
				// if sensor is not repeatable.
				se.log.Warn().Msg("sensor server stopped intentionally")
			} else {
				se.log.Panic().Err(err).Msg("sensor server stopped")
			}
		}
	}()

	// this server reference is used to stop http server in case the sensor is not repeatable and
	// it has done executing trigger/s.
	se.server = srv
}

// validate whether the signal/notification/event is indeed from gateway that this sensor is watching
func (se *sensorExecutionCtx) validateSignal(gatewaySignal *ss_v1alpha1.Event) (*ss_v1alpha1.Signal, bool) {
	for _, signal := range se.sensor.Spec.Signals {
		if signal.Name == gatewaySignal.Context.Source.Host {
			return &signal, true
		}
	}
	return nil, false
}

// Handles signals/notifications/events received from gateways
func (se *sensorExecutionCtx) handleSignals(w http.ResponseWriter, r *http.Request) {
	// parse the request body which contains the cloudevents specification compliant event/signal dispatched from gateway.
	body, err := ioutil.ReadAll(r.Body)
	gatewaySignal := ss_v1alpha1.Event{}
	err = json.Unmarshal(body, &gatewaySignal)
	if err != nil {
		se.log.Error().Err(err).Msg("failed to parse signal received from gateway")
		common.SendErrorResponse(w)
		return
	}

	// validate whether the signal/notification/event is indeed from gateway that this sensor is watching
	selectedSignal, isValidSignal := se.validateSignal(&gatewaySignal)
	if isValidSignal {
		// process the signal/event/notification
		se.queue <- &sensorEventWrapper{
			event:  &gatewaySignal,
			writer: w,
			signal: selectedSignal,
		}
	} else {
		se.log.Warn().Msg("signal from unknown source.")
		common.SendErrorResponse(w)
	}
}

// persist the updates to the Sensor resource
func (se *sensorExecutionCtx) persistUpdates() error {
	var err error
	sensorClient := se.sensorClient.ArgoprojV1alpha1().Sensors(se.sensor.ObjectMeta.Namespace)
	se.sensor, err = sensorClient.Update(se.sensor)
	if err != nil {
		se.log.Warn().Err(err).Msg("error updating sensor")
		if errors.IsConflict(err) {
			return err
		}
		se.log.Info().Msg("re-applying updates on latest version and retrying update")
		err = se.reapplyUpdate()
		if err != nil {
			se.log.Error().Err(err).Msg("failed to re-apply update")
			return err
		}
	}
	se.log.Info().Msg("sensor updated successfully")
	time.Sleep(1 * time.Second)
	return nil
}

// reapply the update to sensor
func (se *sensorExecutionCtx) reapplyUpdate() error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		sClient := se.sensorClient.ArgoprojV1alpha1().Sensors(se.sensor.Namespace)
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

// markNodePhase marks the node with a phase, returns the node
func (se *sensorExecutionCtx) markNodePhase(nodeName string, phase v1alpha1.NodePhase, message ...string) *v1alpha1.NodeStatus {
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
		node.CompletedAt = metav1.MicroTime{Time: time.Now().UTC()}
		se.log.Info().Str("type", string(node.Type)).Str("node-name", node.Name).Msg("completed")
	}
	se.sensor.Status.Nodes[node.ID] = *node
	return node
}
