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
	corev1 "k8s.io/api/core/v1"
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
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/ghodss/yaml"
	"strings"
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
	// controllerInstaceID is the instance id of controller which created this sensor
	controllerInstanceId string
}

// NewsensorExecutionCtx returns a new sensor execution context.
func NewsensorExecutionCtx(sensorClient clientset.Interface, kubeClient kubernetes.Interface,
	clientPool dynamic.ClientPool, discoveryClient discovery.DiscoveryInterface,
	sensor *v1alpha1.Sensor, log zerolog.Logger, wg *sync.WaitGroup, controllerID string) *sensorExecutionCtx {
	return &sensorExecutionCtx{
		sensorClient:    sensorClient,
		kubeClient:      kubeClient,
		clientPool:      clientPool,
		discoveryClient: discoveryClient,
		sensor:          sensor,
		log:             log,
		wg:              wg,
		controllerInstanceId: controllerID,
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

// filters unwanted events
func (se *sensorExecutionCtx) filterK8Event(event *corev1.Event) bool {
	if event.InvolvedObject.Kind == sensor.Kind && event.Source.Component == se.sensor.Name &&
		event.ObjectMeta.Labels[common.LabelEventSeen] == "" &&
		event.ReportingInstance == se.controllerInstanceId &&
		event.ReportingController == se.sensor.Name {
		se.log.Debug().Str("event-name", event.ObjectMeta.Name).Msg("processing k8 event for sensor")
		return true
	}
	se.log.Debug().Str("event-name", event.ObjectMeta.Name).Msg("not processing k8 event for sensor")
	return false
}

// newEventWatcher creates a new event watcher.
func (se *sensorExecutionCtx) newEventWatcher() *cache.ListWatch {
	x := se.kubeClient.CoreV1().RESTClient()
	resource := "events"
	labelSelector := fields.ParseSelectorOrDie(fmt.Sprintf("%s=%s", common.LabelSensorName, se.sensor.Name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector.String()
		req := x.Get().
			Namespace(se.sensor.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.LabelSelector = labelSelector.String()
		options.Watch = true
		req := x.Get().
			Namespace(se.sensor.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// WatchGatewayEvents watches events generated in namespace
func (se *sensorExecutionCtx) WatchSensorEvents(ctx context.Context) (cache.Controller, error) {
	source := se.newEventWatcher()
	_, controller := cache.NewInformer(
		source,
		&corev1.Event{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if newEvent, ok := obj.(*corev1.Event); ok {
					if se.filterK8Event(newEvent) {
						se.log.Info().Msg("detected new k8 Event. Updating sensor resource.")
						err := se.updateSensorResource(newEvent)
						if err != nil {
							se.log.Error().Err(err).Msg("update of sensor resource failed")
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if event, ok := new.(*corev1.Event); ok {
					if se.filterK8Event(event) {
						se.log.Info().Msg("detected k8 Event update. Updating sensor resource.")
						err := se.updateSensorResource(event)
						if err != nil {
							se.log.Error().Err(err).Msg("update of sensor resource failed")
						}
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// updateGatewayResource updates sensor resource
func (se *sensorExecutionCtx) updateSensorResource(event *corev1.Event) error {
	var err error

	defer func() {
		se.log.Info().Str("event-name", event.Name).Msg("marking sensor k8 event as seen")
		// mark event as seen
		event.ObjectMeta.Labels[common.LabelEventSeen] = "true"
		_, err = se.kubeClient.CoreV1().Events(se.sensor.Namespace).Update(event)
		if err != nil {
			se.log.Error().Err(err).Str("event-name", event.ObjectMeta.Name).Msg("failed to mark event as seen")
		}
	}()

	// its better to get latest resource version in case user updated an sensor resource manually
	se.sensor, err = se.sensorClient.ArgoprojV1alpha1().Sensors(se.sensor.Namespace).Get(se.sensor.Name, metav1.GetOptions{})
	if err != nil {
		se.log.Error().Err(err).Str("event-name", event.Name).Msg("failed to retrieve the sensor")
		return err
	}

	// get node/configuration to update
	configurationName, ok := event.ObjectMeta.Labels[common.LabelGatewayConfigurationName]
	if !ok {
		return fmt.Errorf("failed to update sensor resource. no configuration name provided")
	}

	gatewayName, ok := event.ObjectMeta.Labels[common.LabelGatewayName]
	if !ok {
		return fmt.Errorf("failed to update sensor resource. no gateway name provided")
	}

	nodeName := common.DefaultGatewayConfigurationName(gatewayName, configurationName)
	node := getNodeByName(se.sensor, nodeName)
	if node == nil {
		return fmt.Errorf("failed to update sensor resource. node is not initialized. node-name: %s", nodeName)
	}

	se.log.Info().Str("node-id", node.ID).Msg("updating sensor resource...")

	// if this is first time node is being completed, simply mark completed time as that of started time
	// next condition will take care correctly updating complete time
	if ok && node.CompletedAt.Time.IsZero() {
		node.CompletedAt = node.StartedAt
	}
	// check if this event happened after last updated time for the node
	if ok && node.CompletedAt.Time.Before(event.EventTime.Time) {
		node.CompletedAt = event.EventTime
		se.log.Info().Str("node-id", node.ID).Str("action", event.Action).Msg("taking action on configuration")

		switch v1alpha1.NodePhase(event.Action) {
		case v1alpha1.NodePhaseActive, v1alpha1.NodePhaseComplete, v1alpha1.NodePhaseError:
			if signalPayload, ok := event.ObjectMeta.Labels[common.LabelEventForSensorNode]; ok {
				var eventWrapper v1alpha1.EventWrapper
				err := yaml.Unmarshal([]byte(signalPayload), &eventWrapper)
				if err != nil {
					se.log.Error().Err(err).Str("node-id", node.ID).Msg("failed to get event payload through k8 event")
					return err
				}
				node.LatestEvent = &eventWrapper
			}
			se.sensor.Status.Nodes[node.ID] = *node
			se.markNodePhase(node.Name, v1alpha1.NodePhase(event.Action), event.Reason)

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
						return se.persistUpdates()
					}
					// mark trigger as complete.
					se.markNodePhase(trigger.Name, v1alpha1.NodePhaseComplete, "trigger completed successfully")
				}
				if !se.sensor.Spec.Repeat {
					se.server.Shutdown(context.Background())
				}
			}

			return se.persistUpdates()
		default:
			return fmt.Errorf("invalid sensor k8 event. unknown sensor phase, %s", event.Action)
		}
	} else {
		se.log.Warn().Str("node-id", node.ID).Str("node-last-update-time", node.CompletedAt.Time.String()).Str("event-time", event.EventTime.Time.String()).Msg("stale event. skipping update...")
		return nil
	}
}

// getK8Event returns a k8 Event object
func (se *sensorExecutionCtx) getK8Event(reason string, action v1alpha1.NodePhase, gatewayName string, configurationName string) *corev1.Event {
	k8Event := &common.K8Event{
		Namespace: se.sensor.Namespace,
		Name: se.sensor.Name,
		Type: common.LabelSensorStateUpdate,
		Action: string(action),
		ReportingController: se.sensor.Name,
		ReportingInstance: se.controllerInstanceId,
		Labels: map[string]string{
			common.LabelGatewayName: gatewayName,
			common.LabelGatewayConfigurationName: configurationName,
			common.LabelSensorName: se.sensor.Name,
			common.LabelEventSeen: "",
		},
		Kind: sensor.Kind,
		Reason: reason,
	}
	return common.GetK8Event(k8Event)
}

// createK8Event to creates new K8 event in order to persist new state of sensor resource
func (se *sensorExecutionCtx) createK8Event(eventWrapper, nodeName string, errMessage string, err error) (*corev1.Event, error) {
	// create a k8 event to notify event watcher to persist new state of sensor resource.
	// k8 event watching mechanism is used to persist sensor update instead of directly calling persistUpdate method
	// because handleSignals is a invoked as go routine and multiple of such may run at same time causing sensor object to become
	// stale in one of the persistUpdate method which will result in API server complaining resource version being old.
	// K8 events help streamline that process, remove need for adding locks, synchronization etc and most importantly maintains
	// the time consistency of updates.
	gatewayAndConfigurationNames := strings.Split(nodeName, "/")
	var event *corev1.Event
	if err != nil {
		se.log.Error().Err(err).Str("signal", nodeName).Str("phase", string(v1alpha1.NodePhaseError)).Msg(errMessage)
		// create k8 event for error state
		event = se.getK8Event(errMessage, v1alpha1.NodePhaseError, gatewayAndConfigurationNames[0], gatewayAndConfigurationNames[1])
	} else {
		// node successfully completed
		se.log.Info().Str("signal", nodeName).Str("phase", string(v1alpha1.NodePhaseComplete)).Msg("node completed")
		// create k8 event for completion state
		event = se.getK8Event("node completed", v1alpha1.NodePhaseComplete, gatewayAndConfigurationNames[0], gatewayAndConfigurationNames[1])
		event.ObjectMeta.Labels[common.LabelEventForSensorNode] = eventWrapper
	}
	event, err = common.CreateK8Event(event, se.kubeClient)
	if err != nil {
		se.log.Panic().Err(err).Str("signal", nodeName).Msg("failed to create gateway k8 event")
		return nil, err
	}
	se.log.Info().Str("signal", nodeName).Str("phase", event.Action).Str("event-name", event.ObjectMeta.Name).Msg("k8 event created")
	return event, nil
}

// processSignal processes signal received by sensor, validates it, updates the state of the node representing the signal
// and executes trigger if all node with nodetype signal are complete.
func (se *sensorExecutionCtx) processSignal(gatewaySignal *ss_v1alpha1.Event, signal *v1alpha1.Signal, w http.ResponseWriter) {
	var err error
	var errMessage string
	var eventWrapper string

	defer se.createK8Event(eventWrapper, signal.Name, errMessage, err)

	// check if sensor is in error state
	if se.sensor.Status.Phase == v1alpha1.NodePhaseError {
		errMessage = "sensor is in error state. can't process the signal"
		err = fmt.Errorf(errMessage)
		se.log.Warn().Str("signal-src", gatewaySignal.Context.Source.Host).Msg(errMessage)
		common.SendErrorResponse(w)
		return
	}
	se.log.Info().Str("signal-src", gatewaySignal.Context.Source.Host).Msg("signal source")

	// apply filters if any.
	// error is thrown if some problem occurs during filtering the signal
	// apply filters if any
	ok, err := se.filterEvent(signal.Filters, gatewaySignal)
	if err != nil {
		se.log.Error().Err(err).Str("signal-name", gatewaySignal.Context.Source.Host).Err(err).Msg("failed to apply filter")
		errMessage = fmt.Sprintf("error applying filter. Err: %s", err.Error())
		common.SendErrorResponse(w)
		return
	}
	if !ok {
		se.log.Error().Str("signal-name", gatewaySignal.Context.Source.Host).Err(err).Msg("failed to apply filter")
		errMessage = fmt.Sprintf("filter failed")
		common.SendErrorResponse(w)
		return
	}

	if ok {
		// send success response back to gateway as it is a valid notification
		common.SendSuccessResponse(w)
		se.log.Info().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("processing the signal")
		// if signal is already completed, don't process the new event. This will still mark node as complete.
		if getNodeByName(se.sensor, gatewaySignal.Context.Source.Host).Phase == v1alpha1.NodePhaseComplete {
			se.log.Warn().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("signal is already completed")
			return
		}
		// mark this signal/event as seen. this event will be set in sensor node.
		latestEvent := &v1alpha1.EventWrapper{
			Event: *gatewaySignal,
			Seen:  true,
		}
		latestEventBytes, err := yaml.Marshal(latestEvent)
		if err != nil {
			errMessage = fmt.Sprintf("failed to wrap event in EventWrapper. Err: %s", err.Error())
			return
		}
		se.log.Info().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("successfully wrapped event into EventWrapper")
		eventWrapper = string(latestEventBytes)
		return
	} else {
		se.log.Error().Str("signal-name", gatewaySignal.Context.Source.Host).Msg("unknown signal source")
		common.SendErrorResponse(w)
		return
	}
}

// WatchNotifications watches and handles signals sent by the gateway the sensor is interested in.
func (se *sensorExecutionCtx) WatchSignalNotifications() {
	go se.WatchSensorEvents(context.Background())
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
	var isValidSignal bool
	var selectedSignal *v1alpha1.Signal
	for _, signal := range se.sensor.Spec.Signals {
		if signal.Name == gatewaySignal.Context.Source.Host {
			isValidSignal = true
			selectedSignal = &signal
			break
		}
	}

	if isValidSignal {
		// process the signal/event/notification
		se.processSignal(&gatewaySignal, selectedSignal, w)
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
