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
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetNodeByName returns a copy of the node from this sensor for the nodename
// for events this node name should be the name of the event
func GetNodeByName(sensor *v1alpha1.Sensor, nodeName string) *v1alpha1.NodeStatus {
	nodeID := sensor.NodeID(nodeName)
	node, ok := sensor.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return node.DeepCopy()
}

// create a new node
func InitializeNode(sensor *v1alpha1.Sensor, nodeName string, nodeType v1alpha1.NodeType, logger *logrus.Logger, messages ...string) *v1alpha1.NodeStatus {
	if sensor.Status.Nodes == nil {
		sensor.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	}
	nodeID := sensor.NodeID(nodeName)
	oldNode, ok := sensor.Status.Nodes[nodeID]
	if ok {
		logger.WithField(common.LabelNodeName, nodeName).Infoln("node already initialized")
		return &oldNode
	}
	node := v1alpha1.NodeStatus{
		ID:          nodeID,
		Name:        nodeName,
		DisplayName: nodeName,
		Type:        nodeType,
		Phase:       v1alpha1.NodePhaseNew,
		StartedAt:   metav1.MicroTime{Time: time.Now().UTC()},
	}
	if len(messages) > 0 {
		node.Message = messages[0]
	}
	sensor.Status.Nodes[nodeID] = node
	logger.WithFields(
		map[string]interface{}{
			common.LabelNodeType: string(node.Type),
			common.LabelNodeName: node.DisplayName,
			"node-message":       node.Message,
		},
	).Infoln("node is initialized")
	return &node
}

// MarkNodePhase marks the node with a phase, returns the node
func MarkNodePhase(sensor *v1alpha1.Sensor, nodeName string, nodeType v1alpha1.NodeType, phase v1alpha1.NodePhase, event *apicommon.Event, logger *logrus.Logger, message ...string) *v1alpha1.NodeStatus {
	node := GetNodeByName(sensor, nodeName)
	if node.Phase != phase {
		logger.WithFields(
			map[string]interface{}{
				common.LabelNodeType: string(node.Type),
				common.LabelNodeName: node.Name,
				common.LabelPhase:    string(node.Phase),
			},
		).Infoln("marking node phase")
		node.Phase = phase
	}

	if len(message) > 0 {
		node.Message = message[0]
	}

	if nodeType == v1alpha1.NodeTypeEventDependency && event != nil {
		node.Event = event
	}

	if node.Phase == v1alpha1.NodePhaseComplete {
		node.CompletedAt = metav1.MicroTime{Time: time.Now().UTC()}
		logger.WithFields(
			map[string]interface{}{
				common.LabelNodeType: string(node.Type),
				common.LabelNodeName: node.Name,
			},
		).Info("phase marked as completed")
	}

	sensor.Status.Nodes[node.ID] = *node
	return node
}

// AreAllDependenciesResolved checks whether all event dependencies are resolved
func AreAllDependenciesResolved(sensor *v1alpha1.Sensor) bool {
	for _, node := range sensor.Status.Nodes {
		if node.Type == v1alpha1.NodeTypeEventDependency {
			if resolved := IsDependencyResolved(sensor, node.Name); !resolved {
				return false
			}
		}
	}
	return true
}

// IsDependencyResolved checks whether a dependency is resolved.
func IsDependencyResolved(sensor *v1alpha1.Sensor, nodeName string) bool {
	node := GetNodeByName(sensor, nodeName)
	if node.Phase == v1alpha1.NodePhaseError {
		return false
	}
	if node.Phase == v1alpha1.NodePhaseComplete {
		return true
	}
	if &node.StartedAt == nil {
		return false
	}
	if &node.UpdatedAt == nil {
		return false
	}
	resolvedAt := node.ResolvedAt
	if &resolvedAt == nil {
		resolvedAt = node.StartedAt
	}
	if node.UpdatedAt.After(resolvedAt.Time) {
		return true
	}
	return false
}

func MarkUpdatedAt(sensor *v1alpha1.Sensor, nodeName string) *v1alpha1.NodeStatus {
	node := GetNodeByName(sensor, nodeName)
	if node == nil {
		return nil
	}
	if &node.UpdatedAt == nil {
		return node
	}
	node.UpdatedAt.Time = time.Now().UTC()
	sensor.Status.Nodes[node.ID] = *node
	return node
}

func MarkResolvedAt(sensor *v1alpha1.Sensor, nodeName string) *v1alpha1.NodeStatus {
	node := GetNodeByName(sensor, nodeName)
	if node == nil {
		return nil
	}
	if &node.ResolvedAt == nil {
		return node
	}
	node.ResolvedAt.Time = time.Now().UTC()
	sensor.Status.Nodes[node.ID] = *node
	return node
}
