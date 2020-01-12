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
package v1alpha1

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	logger = common.NewArgoEventsLogger()
)

// HasLocation whether or not an minio has a location defined
func (a *ArtifactLocation) HasLocation() bool {
	return a.S3 != nil || a.Inline != nil || a.File != nil || a.URL != nil
}

// IsComplete determines if the node has reached an end state
func (node NodeStatus) IsComplete() bool {
	return node.Phase == NodePhaseComplete ||
		node.Phase == NodePhaseError
}

// IsComplete determines if the sensor has reached an end state
func (s *Sensor) IsComplete() bool {
	if !(s.Status.Phase == NodePhaseComplete || s.Status.Phase == NodePhaseError) {
		return false
	}
	for _, node := range s.Status.Nodes {
		if !node.IsComplete() {
			return false
		}
	}
	return true
}

// GetNodeByName returns the node from this sensor for the nodename
// for events this node name should be the name of the event
func (s *Sensor) GetNodeByName(nodeName string) *NodeStatus {
	nodeID := s.NodeID(nodeName)
	node, ok := s.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

// create a new node
func (s *Sensor) InitializeNode(nodeName string, nodeType NodeType, logger *logrus.Logger, messages ...string) *NodeStatus {
	if s.Status.Nodes == nil {
		s.Status.Nodes = make(map[string]NodeStatus)
	}
	nodeID := s.NodeID(nodeName)
	oldNode, ok := s.Status.Nodes[nodeID]
	if ok {
		logger.WithField(common.LabelNodeName, nodeName).Infoln("node already initialized")
		return &oldNode
	}
	node := NodeStatus{
		ID:          nodeID,
		Name:        nodeName,
		DisplayName: nodeName,
		Type:        nodeType,
		Phase:       NodePhaseNew,
		StartedAt:   metav1.MicroTime{Time: time.Now().UTC()},
	}
	if len(messages) > 0 {
		node.Message = messages[0]
	}
	s.Status.Nodes[nodeID] = node
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
func (s *Sensor) MarkNodePhase(nodeName string, nodeType NodeType, phase NodePhase, event *apicommon.Event, logger *logrus.Logger, message ...string) *NodeStatus {
	node := s.GetNodeByName(nodeName)
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

	if nodeType == NodeTypeEventDependency && event != nil {
		node.Event = event
	}

	if node.Phase == NodePhaseComplete {
		node.CompletedAt = metav1.MicroTime{Time: time.Now().UTC()}
		logger.WithFields(
			map[string]interface{}{
				common.LabelNodeType: string(node.Type),
				common.LabelNodeName: node.Name,
			},
		).Info("phase marked as completed")
	}

	s.Status.Nodes[node.ID] = *node
	return node
}

// IsDependencyResolved checks if a dependency is resolved. If so, it updates the resolveAt time for the node.
func (s *Sensor) IsDependencyResolved(node *NodeStatus) bool {
	if node.Phase == NodePhaseError {
		return false
	}
	// this means the dependency is already resolved
	if node.Phase == NodePhaseComplete {
		return true
	}
	if &node.StartedAt == nil {
		return false
	}
	resolvedAt := node.ResolvedAt
	if &resolvedAt == nil {
		resolvedAt = node.StartedAt
	}
	updatedAt := node.UpdatedAt
	// this should not happen
	if &updatedAt == nil {
		return false
	}

	if updatedAt.After(resolvedAt.Time) {
		node.ResolvedAt.Time = time.Now().UTC()
		node = s.MarkNodeResolveTime(node)
		s.MarkNodePhase(node.Name, NodeTypeEventDependency, NodePhaseComplete, nil, logger)
		return true
	}

	return false
}

// AreAllNodesSuccess determines if all nodes of the given type have completed successfully
func (s *Sensor) AreAllNodesSuccess(nodeType NodeType) bool {
	for _, node := range s.Status.Nodes {
		if node.Type == nodeType && node.Phase != NodePhaseComplete {
			return false
		}
	}
	return true
}

// NodeID creates a deterministic node ID based on a node name
// we support 3 kinds of "nodes" - sensors, events, triggers
// each should pass it's name field
func (s *Sensor) NodeID(name string) string {
	if name == s.ObjectMeta.Name {
		return s.ObjectMeta.Name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return fmt.Sprintf("%s-%v", s.ObjectMeta.Name, h.Sum32())
}

// MarkNodeResolveTime marks the time at which node was resolved.
func (s *Sensor) MarkNodeResolveTime(node *NodeStatus) *NodeStatus {
	if node == nil {
		return node
	}
	node.ResolvedAt = metav1.MicroTime{Time: time.Now()}
	s.Status.Nodes[node.ID] = *node
	return node
}

// MarkNodeResolveTime marks the time at which node was updated.
func (s *Sensor) MarkNodeUpdateTime(node *NodeStatus) *NodeStatus {
	if node == nil {
		return nil
	}
	node.UpdatedAt = metav1.MicroTime{Time: time.Now()}
	s.Status.Nodes[node.ID] = *node
	return node
}
