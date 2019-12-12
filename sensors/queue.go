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
	"github.com/argoproj/argo-events/common"
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// processNotificationQueue processes event received on internal queue and updates the state of the node representing the event dependency
func (sensorCtx *sensorContext) processNotificationQueue(notification *notification) {
	defer func() {
		// persist updates to sensor resource
		labels := map[string]string{
			common.LabelSensorName:           sensorCtx.sensor.Name,
			snctrl.LabelPhase:                string(sensorCtx.sensor.Status.Phase),
			snctrl.LabelControllerInstanceID: sensorCtx.controllerInstanceID,
			common.LabelOperation:            "persist_state_update",
		}
		eventType := common.StateChangeEventType

		updatedSensor, err := snctrl.PersistUpdates(sensorCtx.sensorClient, sensorCtx.sensor, sensorCtx.logger)
		if err != nil {
			sensorCtx.logger.WithError(err).Error("failed to persist sensor update, escalating...")
			eventType = common.EscalationEventType
		}

		// update sensor ref. in case of failure to persist updates, this is a deep copy of old sensor resource
		sensorCtx.sensor = updatedSensor

		labels[common.LabelEventType] = string(eventType)
		if err := common.GenerateK8sEvent(sensorCtx.kubeClient, "persist update", eventType, "sensor resource update", sensorCtx.sensor.Name,
			sensorCtx.sensor.Namespace, sensorCtx.controllerInstanceID, sensor.Kind, labels); err != nil {
			sensorCtx.logger.WithError(err).Error("failed to create K8s event to logger sensor resource persist operation")
			return
		}

		sensorCtx.logger.Info("successfully persisted sensor resource update and created K8s event")
	}()

	switch notification.notificationType {
	case v1alpha1.ResourceUpdateNotification:
		sensorCtx.logger.Info("sensor resource update")
		// update sensor resource
		sensorCtx.sensor = notification.sensor

		// initialize new dependencies
		for _, dependency := range sensorCtx.sensor.Spec.Dependencies {
			if node := snctrl.GetNodeByName(sensorCtx.sensor, dependency.Name); node == nil {
				snctrl.InitializeNode(sensorCtx.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, sensorCtx.logger)
				snctrl.MarkNodePhase(sensorCtx.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, sensorCtx.logger, "dependency is active")
			}
		}

		// initialize new dependency groups
		for _, group := range sensorCtx.sensor.Spec.DependencyGroups {
			if node := snctrl.GetNodeByName(sensorCtx.sensor, group.Name); node == nil {
				snctrl.InitializeNode(sensorCtx.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, sensorCtx.logger)
				snctrl.MarkNodePhase(sensorCtx.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, sensorCtx.logger, "dependency group is active")
			}
		}

		// initialize new triggers
		for _, t := range sensorCtx.sensor.Spec.Triggers {
			if node := snctrl.GetNodeByName(sensorCtx.sensor, t.Template.Name); node == nil {
				snctrl.InitializeNode(sensorCtx.sensor, t.Template.Name, v1alpha1.NodeTypeTrigger, sensorCtx.logger)
			}
		}

		sensorCtx.deleteStaleStatusNodes()

	default:
		sensorCtx.logger.WithField("notification-type", string(notification.notificationType)).Error("unknown notification type")
	}
}

func (sensorCtx *sensorContext) deleteStaleStatusNodes() {
	// delete old status nodes if any
statusNodes:
	for _, statusNode := range sensorCtx.sensor.Status.Nodes {
		for _, dep := range sensorCtx.sensor.Spec.Dependencies {
			if statusNode.Type == v1alpha1.NodeTypeEventDependency && dep.Name == statusNode.Name {
				continue statusNodes
			}
		}
		for _, depGroup := range sensorCtx.sensor.Spec.DependencyGroups {
			if statusNode.Type == v1alpha1.NodeTypeDependencyGroup && depGroup.Name == statusNode.Name {
				continue statusNodes
			}
		}
		for _, trigger := range sensorCtx.sensor.Spec.Triggers {
			if statusNode.Type == v1alpha1.NodeTypeTrigger && trigger.Template.Name == statusNode.Name {
				continue statusNodes
			}
		}
		// corresponding node not found in spec. deleting status node
		sensorCtx.logger.WithField("status-node", statusNode.Name).Info("deleting old status node")
		nodeId := sensorCtx.sensor.NodeID(statusNode.Name)
		delete(sensorCtx.sensor.Status.Nodes, nodeId)
	}
}
