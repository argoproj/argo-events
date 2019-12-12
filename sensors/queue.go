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

// processNotificationQueue processes Event received on internal queue and updates the state of the node representing the Event dependency
func (sensorCtx *SensorContext) processNotificationQueue(notification *Notification) {
	defer func() {
		// persist updates to Sensor resource
		labels := map[string]string{
			common.LabelSensorName:           sensorCtx.Sensor.Name,
			snctrl.LabelPhase:                string(sensorCtx.Sensor.Status.Phase),
			snctrl.LabelControllerInstanceID: sensorCtx.ControllerInstanceID,
			common.LabelOperation:            "persist_state_update",
		}
		eventType := common.StateChangeEventType

		updatedSensor, err := snctrl.PersistUpdates(sensorCtx.SensorClient, sensorCtx.Sensor, sensorCtx.Logger)
		if err != nil {
			sensorCtx.Logger.WithError(err).Error("failed to persist Sensor update, escalating...")
			eventType = common.EscalationEventType
		}

		// update Sensor ref. in case of failure to persist updates, this is a deep copy of old Sensor resource
		sensorCtx.Sensor = updatedSensor

		labels[common.LabelEventType] = string(eventType)
		if err := common.GenerateK8sEvent(sensorCtx.KubeClient, "persist update", eventType, "Sensor resource update", sensorCtx.Sensor.Name,
			sensorCtx.Sensor.Namespace, sensorCtx.ControllerInstanceID, sensor.Kind, labels); err != nil {
			sensorCtx.Logger.WithError(err).Error("failed to create K8s Event to Logger Sensor resource persist operation")
			return
		}

		sensorCtx.Logger.Info("successfully persisted Sensor resource update and created K8s Event")
	}()

	switch notification.NotificationType {
	case v1alpha1.ResourceUpdateNotification:
		sensorCtx.Logger.Info("Sensor resource update")
		// update Sensor resource
		sensorCtx.Sensor = notification.Sensor

		// initialize new dependencies
		for _, dependency := range sensorCtx.Sensor.Spec.Dependencies {
			if node := snctrl.GetNodeByName(sensorCtx.Sensor, dependency.Name); node == nil {
				snctrl.InitializeNode(sensorCtx.Sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, sensorCtx.Logger)
				snctrl.MarkNodePhase(sensorCtx.Sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, sensorCtx.Logger, "dependency is active")
			}
		}

		// initialize new dependency groups
		for _, group := range sensorCtx.Sensor.Spec.DependencyGroups {
			if node := snctrl.GetNodeByName(sensorCtx.Sensor, group.Name); node == nil {
				snctrl.InitializeNode(sensorCtx.Sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, sensorCtx.Logger)
				snctrl.MarkNodePhase(sensorCtx.Sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, sensorCtx.Logger, "dependency group is active")
			}
		}

		// initialize new triggers
		for _, t := range sensorCtx.Sensor.Spec.Triggers {
			if node := snctrl.GetNodeByName(sensorCtx.Sensor, t.Template.Name); node == nil {
				snctrl.InitializeNode(sensorCtx.Sensor, t.Template.Name, v1alpha1.NodeTypeTrigger, sensorCtx.Logger)
			}
		}

		sensorCtx.deleteStaleStatusNodes()

	default:
		sensorCtx.Logger.WithField("Notification-type", string(notification.NotificationType)).Error("unknown Notification type")
	}
}

func (sensorCtx *SensorContext) deleteStaleStatusNodes() {
	// delete old status nodes if any
statusNodes:
	for _, statusNode := range sensorCtx.Sensor.Status.Nodes {
		for _, dep := range sensorCtx.Sensor.Spec.Dependencies {
			if statusNode.Type == v1alpha1.NodeTypeEventDependency && dep.Name == statusNode.Name {
				continue statusNodes
			}
		}
		for _, depGroup := range sensorCtx.Sensor.Spec.DependencyGroups {
			if statusNode.Type == v1alpha1.NodeTypeDependencyGroup && depGroup.Name == statusNode.Name {
				continue statusNodes
			}
		}
		for _, trigger := range sensorCtx.Sensor.Spec.Triggers {
			if statusNode.Type == v1alpha1.NodeTypeTrigger && trigger.Template.Name == statusNode.Name {
				continue statusNodes
			}
		}
		// corresponding node not found in spec. deleting status node
		sensorCtx.Logger.WithField("status-node", statusNode.Name).Info("deleting old status node")
		nodeId := sensorCtx.Sensor.NodeID(statusNode.Name)
		delete(sensorCtx.Sensor.Status.Nodes, nodeId)
	}
}
