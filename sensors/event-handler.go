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

package sensors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/common"
	sn "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	ss_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// simple check to make sure two event dependencies list are equal
func EqualEventDependencies(a, b []v1alpha1.EventDependency) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// simple check to make sure two triggers list are equal
func EqualTriggers(a, b []v1alpha1.Trigger) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// processUpdateNotification processes event received by sensor, validates it, updates the state of the node representing the event dependency
func (sec *sensorExecutionCtx) processUpdateNotification(ew *updateNotification) {
	defer func() {
		// persist updates to sensor resource
		labels := map[string]string{
			common.LabelSensorName:                    sec.sensor.Name,
			common.LabelSensorKeyPhase:                string(sec.sensor.Status.Phase),
			common.LabelKeySensorControllerInstanceID: sec.controllerInstanceID,
			common.LabelOperation:                     "persist_state_update",
		}
		eventType := common.StateChangeEventType

		updatedSensor, err := sn.PersistUpdates(sec.sensorClient, sec.sensor, sec.controllerInstanceID, &sec.log)
		if err != nil {
			sec.log.Error().Err(err).Msg("failed to persist sensor update, escalating...")
			// escalate failure
			eventType = common.EscalationEventType
		}

		// update sensor ref. in case of failure to persist updates, this is a deep copy of old sensor resource
		sec.sensor = updatedSensor

		labels[common.LabelEventType] = string(eventType)
		if err := common.GenerateK8sEvent(sec.kubeClient, "persist update", eventType, "sensor resource update", sec.sensor.Name,
			sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
			sec.log.Error().Err(err).Msg("failed to create K8s event to log sensor resource persist operation")
			return
		}
		sec.log.Info().Msg("successfully persisted sensor resource update and created K8s event")
	}()

	switch ew.notificationType {
	case v1alpha1.EventNotification:
		sec.log.Info().Str("event-dependency-name", ew.event.Context.Source.Host).Msg("received event notification")

		// apply filters if any.
		ok, err := sec.filterEvent(ew.eventDependency.Filters, ew.event)
		if err != nil {
			sec.log.Error().Err(err).Str("event-dependency-name", ew.event.Context.Source.Host).Err(err).Msg("failed to apply filter")

			// escalate error
			labels := map[string]string{
				common.LabelEventType:   string(common.EscalationEventType),
				common.LabelEventSource: ew.event.Context.Source.Host,
				common.LabelSensorName:  sec.sensor.Name,
				common.LabelOperation:   "filter_event",
			}
			if err := common.GenerateK8sEvent(sec.kubeClient, "apply filter failed", common.OperationFailureEventType, "filtering event", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
				sec.log.Error().Err(err).Msg("failed to create K8s event to log filtering error")
			}

			// change node state to error
			sn.MarkNodePhase(sec.sensor, ew.event.Context.Source.Host, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseError, nil, &sec.log, fmt.Sprintf("failed to apply filter. err: %v", err))

			common.SendErrorResponse(ew.writer)
			return
		}

		// event is not valid
		if !ok {
			sec.log.Error().Str("event-dependency-name", ew.event.Context.Source.Host).Msg("event did not pass filters")

			// change node state to error
			sn.MarkNodePhase(sec.sensor, ew.event.Context.Source.Host, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseError, nil, &sec.log, "event did not pass filters")

			common.SendErrorResponse(ew.writer)
			return
		}

		// send success response back to gateway as it is a valid notification
		common.SendSuccessResponse(ew.writer)
		sn.MarkNodePhase(sec.sensor, ew.event.Context.Source.Host, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, ew.event, &sec.log, "event is received")

		// check if all event dependencies are complete and kick-off triggers
		sec.processTriggers()

	case v1alpha1.ResourceUpdateNotification:
		sec.log.Info().Msg("sensor resource update")
		// update event dependencies
		if !EqualEventDependencies(sec.sensor.Spec.EventDependencies, ew.sensor.Spec.EventDependencies) {
			sec.sensor.Spec.EventDependencies = ew.sensor.Spec.EventDependencies

			// initialize new event dependencies
			for _, ed := range sec.sensor.Spec.EventDependencies {
				if node := sn.GetNodeByName(sec.sensor, ed.Name); node == nil {
					sn.InitializeNode(sec.sensor, ed.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, &sec.log)
				}
			}
		}

		// update triggers
		if !EqualTriggers(sec.sensor.Spec.Triggers, ew.sensor.Spec.Triggers) {
			sec.sensor.Spec.Triggers = ew.sensor.Spec.Triggers

			// initialize new triggers
			for _, t := range sec.sensor.Spec.Triggers {
				if node := sn.GetNodeByName(sec.sensor, t.Name); node == nil {
					sn.InitializeNode(sec.sensor, t.Name, v1alpha1.NodeTypeTrigger, v1alpha1.NodePhaseNew, &sec.log)
				}
			}
		}

	default:
		sec.log.Error().Str("notification-type", string(ew.notificationType)).Msg("unknown notification type")
	}
}

// WatchEventsFromGateways watches and handles events received from the gateway.
func (sec *sensorExecutionCtx) WatchEventsFromGateways() {
	// start processing the update notification queue
	go func() {
		for e := range sec.queue {
			sec.processUpdateNotification(e)
		}
	}()

	// if a sensor resource is updated, syncSensor will take care of updating current sensor context
	// note: updating sensor resource will result in loss of all existing events stored in nodes.
	// Once the NATS streaming support is added, user can replay the events.
	go func() {
		if _, err := sec.syncSensor(context.Background()); err != nil {
			sec.log.Error().Err(err).Msg("failed to sync sensor resource")
		}
	}()

	// create a http server. this server listens for events from gateway.
	sec.server = &http.Server{Addr: fmt.Sprintf(":%s", common.SensorServicePort)}

	// add a handler to handle incoming events
	http.HandleFunc("/", sec.eventHandler)

	sec.log.Info().Str("port", string(common.SensorServicePort)).Msg("sensor started listening")
	if err := sec.server.ListenAndServe(); err != nil {
		sec.log.Error().Err(err).Msg("sensor server stopped")
		// escalate error
		labels := map[string]string{
			common.LabelEventType:  string(common.EscalationEventType),
			common.LabelSensorName: sec.sensor.Name,
			common.LabelOperation:  "server_shutdown",
		}
		if err := common.GenerateK8sEvent(sec.kubeClient, fmt.Sprintf("sensor server stopped"), common.EscalationEventType,
			"server shutdown", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
			sec.log.Error().Err(err).Msg("failed to create K8s event to log server shutdown error")
		}
	}
}

// Handles events received from gateways
func (sec *sensorExecutionCtx) eventHandler(w http.ResponseWriter, r *http.Request) {
	sec.log.Info().Msg("received an event from gateway")
	body, err := ioutil.ReadAll(r.Body)
	var event *ss_v1alpha1.Event
	if err = json.Unmarshal(body, &event); err != nil {
		sec.log.Error().Err(err).Msg("failed to parse event received from gateway")
		common.SendErrorResponse(w)
		return
	}

	// validate whether the event is from gateway that this sensor is watching
	if eventDependency, isValidEvent := sec.validateEvent(event); isValidEvent {
		// process the event
		sec.queue <- &updateNotification{
			event:            event,
			writer:           w,
			eventDependency:  eventDependency,
			notificationType: v1alpha1.EventNotification,
		}
		return
	}

	sec.log.Warn().Str("event-source-name", event.Context.Source.Host).Msg("event is from unknown source")
	common.SendErrorResponse(w)
}

// validateEvent validates whether the event is indeed from gateway that this sensor is watching
func (sec *sensorExecutionCtx) validateEvent(events *ss_v1alpha1.Event) (*ss_v1alpha1.EventDependency, bool) {
	for _, event := range sec.sensor.Spec.EventDependencies {
		if event.Name == events.Context.Source.Host {
			return &event, true
		}
	}
	return nil, false
}
