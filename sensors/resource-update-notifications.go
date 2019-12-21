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
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/types"
)

// OperateResourceUpdateNotification operates on the resource state notification update
func (sensorCtx *SensorContext) operateResourceUpdateNotification(notification *types.Notification) {
	sensorCtx.Logger.Info("Sensor resource update")
	// update Sensor resource
	sensorCtx.Sensor = notification.Sensor.DeepCopy()

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

	// delete stale nodes
	nodes := deleteStaleStatusNodes(sensorCtx.Sensor)
	if nodes == nil {
		return
	}
	for _, node := range nodes {
		sensorCtx.Logger.WithField("status-node", node.Name).Info("deleting old status node")
		nodeId := sensorCtx.Sensor.NodeID(node.Name)
		delete(sensorCtx.Sensor.Status.Nodes, nodeId)
	}
}

// deleteStaleStatusNodes deletes old status nodes if any
func deleteStaleStatusNodes(sensor *v1alpha1.Sensor) []v1alpha1.NodeStatus {
	var nodes []v1alpha1.NodeStatus
statusNodes:
	for _, node := range sensor.Status.Nodes {
		for _, dep := range sensor.Spec.Dependencies {
			if node.Type == v1alpha1.NodeTypeEventDependency && dep.Name == node.Name {
				continue statusNodes
			}
		}
		for _, depGroup := range sensor.Spec.DependencyGroups {
			if node.Type == v1alpha1.NodeTypeDependencyGroup && depGroup.Name == node.Name {
				continue statusNodes
			}
		}
		for _, trigger := range sensor.Spec.Triggers {
			if node.Type == v1alpha1.NodeTypeTrigger && trigger.Template.Name == node.Name {
				continue statusNodes
			}
		}
		nodes = append(nodes, node)
	}
	return nodes
}
