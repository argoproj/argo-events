/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sensor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	ss_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/rs/zerolog"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
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
	// queue is internal queue to manage incoming events
	queue chan *eventWrapper
	// controllerInstanceID is the instance ID of sensor controller processing this sensor
	controllerInstanceID string
}

// eventWrapper is a wrapper around event received from gateway and the event dependency
type eventWrapper struct {
	event           *ss_v1alpha1.Event
	eventDependency *v1alpha1.EventDependency
	writer          http.ResponseWriter
}

// NewSensorExecutionCtx returns a new sensor execution context.
func NewSensorExecutionCtx(sensorClient clientset.Interface, kubeClient kubernetes.Interface,
	clientPool dynamic.ClientPool, discoveryClient discovery.DiscoveryInterface,
	sensor *v1alpha1.Sensor, log zerolog.Logger, controllerInstanceID string) *sensorExecutionCtx {
	return &sensorExecutionCtx{
		sensorClient:         sensorClient,
		kubeClient:           kubeClient,
		clientPool:           clientPool,
		discoveryClient:      discoveryClient,
		sensor:               sensor,
		log:                  log,
		queue:                make(chan *eventWrapper),
		controllerInstanceID: controllerInstanceID,
	}
}

// processEvent processes event received by sensor, validates it, updates the state of the node representing the event dependency
func (sec *sensorExecutionCtx) processEvent(ew *eventWrapper) {
	sec.log.Info().Str("event-src", ew.event.Context.Source.Host).Msg("event source")

	// apply filters if any.
	ok, err := sec.filterEvent(ew.eventDependency.Filters, ew.event)
	if err != nil {
		sec.log.Error().Err(err).Str("event-dependency-name", ew.event.Context.Source.Host).Err(err).Msg("failed to apply filter")
		common.SendErrorResponse(ew.writer)
		return
	}
	if !ok {
		sec.log.Error().Str("event-dependency-name", ew.event.Context.Source.Host).Err(err).Msg("failed to apply filter")
		common.SendErrorResponse(ew.writer)
		return
	}

	// send success response back to gateway as it is a valid notification
	common.SendSuccessResponse(ew.writer)

	node := sensor.GetNodeByName(sec.sensor, ew.event.Context.Source.Host)
	// mark this eventDependency/event as seen. this event will be set in sensor node.
	node.Event = ew.event
	sec.sensor.Status.Nodes[node.ID] = *node
	sec.processTriggers()
	return
}

// WatchNotifications watches and handles events received from the gateway.
func (sec *sensorExecutionCtx) WatchGatewayEvents() {
	// start processing the event queue
	go func() {
		for {
			e := <-sec.queue
			sec.processEvent(e)
		}
	}()

	// create a http server. this server listens for events from gateway.
	sec.server = &http.Server{Addr: fmt.Sprintf(":%s", common.SensorServicePort)}

	// add a handler to handle incoming events
	http.HandleFunc("/", sec.eventHandler)

	sec.log.Info().Str("port", string(common.SensorServicePort)).Msg("sensor started listening")
	if err := sec.server.ListenAndServe(); err != nil {
		sec.log.Error().Err(err).Msg("sensor server stopped")
		// todo: escalate using K8s event
		// this K8s event will be used by controller to mark sensor resource as error.
	}
}

// Handles events received from gateways
func (sec *sensorExecutionCtx) eventHandler(w http.ResponseWriter, r *http.Request) {
	// parse the request body which contains the cloudevents specification compliant event received from gateway.
	sec.log.Info().Msg("received an event from gateway")
	body, err := ioutil.ReadAll(r.Body)
	var event *ss_v1alpha1.Event
	err = json.Unmarshal(body, &event)
	if err != nil {
		sec.log.Error().Err(err).Msg("failed to parse event received from gateway")
		common.SendErrorResponse(w)
		return
	}

	// validate whether the event is indeed from gateway that this sensor is watching
	eventDependency, isValidSignal := sec.validateEvent(event)
	if isValidSignal {
		// process the event
		sec.queue <- &eventWrapper{
			event:           event,
			writer:          w,
			eventDependency: eventDependency,
		}
	} else {
		sec.log.Warn().Msg("eventDependency from unknown source.")
		common.SendErrorResponse(w)
	}
}

// validate whether the event is indeed from gateway that this sensor is watching
func (sec *sensorExecutionCtx) validateEvent(events *ss_v1alpha1.Event) (*ss_v1alpha1.EventDependency, bool) {
	for _, event := range sec.sensor.Spec.EventDependencies {
		if event.Name == events.Context.Source.Host {
			return &event, true
		}
	}
	return nil, false
}
