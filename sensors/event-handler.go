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
	"net/http"

	"github.com/argoproj/argo-events/common"
	sn "github.com/argoproj/argo-events/controllers/sensor"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	ss_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/gobwas/glob"
)

// processUpdateNotification processes event received by sensor, validates it, updates the state of the node representing the event dependency
func (sec *sensorExecutionCtx) processUpdateNotification(ew *updateNotification) {
	defer func() {
		// persist updates to sensor resource
		labels := map[string]string{
			common.LabelSensorName:       sec.sensor.Name,
			sn.LabelPhase:                string(sec.sensor.Status.Phase),
			sn.LabelControllerInstanceID: sec.controllerInstanceID,
			common.LabelOperation:        "persist_state_update",
		}
		eventType := common.StateChangeEventType

		updatedSensor, err := sn.PersistUpdates(sec.sensorClient, sec.sensor, sec.log)
		if err != nil {
			sec.log.WithError(err).Error("failed to persist sensor update, escalating...")
			// escalate failure
			eventType = common.EscalationEventType
		}

		// update sensor ref. in case of failure to persist updates, this is a deep copy of old sensor resource
		sec.sensor = updatedSensor

		labels[common.LabelEventType] = string(eventType)
		if err := common.GenerateK8sEvent(sec.kubeClient, "persist update", eventType, "sensor resource update", sec.sensor.Name,
			sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
			sec.log.WithError(err).Error("failed to create K8s event to log sensor resource persist operation")
			return
		}

		sec.log.Info("successfully persisted sensor resource update and created K8s event")
	}()

	switch ew.notificationType {
	case v1alpha1.EventNotification:
		nodeName := ew.eventDependency.Name
		log := sec.log.WithField(common.LabelEventSource, ew.event.Context.Source.Host)
		log.Info("received event notification")

		// apply filters if any.
		ok, err := sec.filterEvent(ew.eventDependency.Filters, ew.event)
		if err != nil {
			log.WithError(err).Error("failed to apply filter")

			// escalate error
			labels := map[string]string{
				common.LabelEventType:   string(common.EscalationEventType),
				common.LabelEventSource: ew.event.Context.Source.Host,
				common.LabelSensorName:  sec.sensor.Name,
				common.LabelOperation:   "filter_event",
			}
			if err := common.GenerateK8sEvent(sec.kubeClient, "apply filter failed", common.OperationFailureEventType, "filtering event", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
				log.WithError(err).Error("failed to create K8s event to log filtering error")
			}

			// change node state to error
			sn.MarkNodePhase(sec.sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseError, nil, sec.log, fmt.Sprintf("failed to apply filter. err: %v", err))
			return
		}

		// event is not valid
		if !ok {
			log.Error("event did not pass filters")

			// change node state to error
			sn.MarkNodePhase(sec.sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseError, nil, sec.log, "event did not pass filters")
			return
		}

		sn.MarkNodePhase(sec.sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, ew.event, sec.log, "event is received")

		// check if triggers can be processed and executed
		canProcess, err := sec.canProcessTriggers()
		if err != nil {
			sec.log.WithError(err).Error("error occurred while determining triggers can be processed")
			return
		}
		if !canProcess {
			sec.log.Warn("triggers can't be processed at this time, won't fire triggers")
			return
		}

		// triggers are ready to process
		sec.processTriggers()

	case v1alpha1.ResourceUpdateNotification:
		sec.log.Info("sensor resource update")
		// update sensor resource
		sec.sensor = ew.sensor

		hasDependenciesUpdated := false

		// initialize new dependencies
		for _, dependency := range sec.sensor.Spec.Dependencies {
			if node := sn.GetNodeByName(sec.sensor, dependency.Name); node == nil {
				sn.InitializeNode(sec.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, sec.log)
				sn.MarkNodePhase(sec.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, sec.log, "dependency is active")
			}
		}

		// initialize new dependency groups
		for _, group := range sec.sensor.Spec.DependencyGroups {
			if node := sn.GetNodeByName(sec.sensor, group.Name); node == nil {
				sn.InitializeNode(sec.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, sec.log)
				sn.MarkNodePhase(sec.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, sec.log, "dependency group is active")
			}
		}

		// initialize new triggers
		for _, t := range sec.sensor.Spec.Triggers {
			if node := sn.GetNodeByName(sec.sensor, t.Template.Name); node == nil {
				hasDependenciesUpdated = true
				sn.InitializeNode(sec.sensor, t.Template.Name, v1alpha1.NodeTypeTrigger, sec.log)
			}
		}

		sec.deleteStaleStatusNodes()

		if hasDependenciesUpdated {
			sec.NatsEventProtocol()
		}

	default:
		sec.log.WithField("notification-type", string(ew.notificationType)).Error("unknown notification type")
	}
}

func (sec *sensorExecutionCtx) deleteStaleStatusNodes() {
	// delete old status nodes if any
statusNodes:
	for _, statusNode := range sec.sensor.Status.Nodes {
		for _, dep := range sec.sensor.Spec.Dependencies {
			if statusNode.Type == v1alpha1.NodeTypeEventDependency && dep.Name == statusNode.Name {
				continue statusNodes
			}
		}
		for _, depGroup := range sec.sensor.Spec.DependencyGroups {
			if statusNode.Type == v1alpha1.NodeTypeDependencyGroup && depGroup.Name == statusNode.Name {
				continue statusNodes
			}
		}
		for _, trigger := range sec.sensor.Spec.Triggers {
			if statusNode.Type == v1alpha1.NodeTypeTrigger && trigger.Template.Name == statusNode.Name {
				continue statusNodes
			}
		}
		// corresponding node not found in spec. deleting status node
		sec.log.WithField("status-node", statusNode.Name).Info("deleting old status node")
		nodeId := sec.sensor.NodeID(statusNode.Name)
		delete(sec.sensor.Status.Nodes, nodeId)
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

	// sync sensor resource after updates
	go sec.syncSensor(context.Background())

	switch sec.sensor.Spec.EventProtocol.Type {
	case pc.HTTP:
		sec.HttpEventProtocol()
	case pc.NATS:
		sec.NatsEventProtocol()
		var err error
		if sec.sensor, err = sn.PersistUpdates(sec.sensorClient, sec.sensor, sec.log); err != nil {
			sec.log.WithError(err).Error("failed to persist sensor update")
			labels := map[string]string{
				common.LabelEventType:  string(common.OperationFailureEventType),
				common.LabelSensorName: sec.sensor.Name,
				common.LabelOperation:  "persist_after_nats_conn_update_failure",
			}
			if err := common.GenerateK8sEvent(sec.kubeClient, "persist updates nats connection update failed", common.OperationFailureEventType, "persist updates nats connection update", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
				sec.log.WithError(err).Error("failed to create K8s event to log persist updates nats connection update failure")
			}
		}
		select {}
	}
}

// validateEvent validates whether the event is indeed from gateway that this sensor is watching
func (sec *sensorExecutionCtx) validateEvent(events *apicommon.Event) (*ss_v1alpha1.EventDependency, bool) {
	for _, dependency := range sec.sensor.Spec.Dependencies {
		g, err := glob.Compile(dependency.Name)
		if err != nil {
			sec.log.WithError(err).Error("invalid glob in dependency name")
			continue
		}
		if g.Match(events.Context.Source.Host) {
			return &dependency, true
		}
	}
	return nil, false
}

func (sec *sensorExecutionCtx) parseEvent(payload []byte) (*apicommon.Event, error) {
	var event *apicommon.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		response := "failed to parse event received from gateway"
		sec.log.WithError(err).Error(response)
		return nil, err
	}
	return event, nil
}

func (sec *sensorExecutionCtx) sendEventToInternalQueue(event *apicommon.Event, writer http.ResponseWriter) bool {
	// validate whether the event is from gateway that this sensor is watching
	if eventDependency, isValidEvent := sec.validateEvent(event); isValidEvent {
		// process the event
		sec.queue <- &updateNotification{
			event:            event,
			writer:           writer,
			eventDependency:  eventDependency,
			notificationType: v1alpha1.EventNotification,
		}
		return true
	}
	return false
}
