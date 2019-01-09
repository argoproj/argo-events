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

package gateways

import (
	"fmt"
	gtw "github.com/argoproj/argo-events/controllers/gateway"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/gateway"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventSourceStatus encapsulates state of an event source
type EventSourceStatus struct {
	// Id of the event source
	Id string
	// Name of the event source
	Name string
	// Message
	Message string
	// Phase of the event source
	Phase v1alpha1.NodePhase
	// Gateway reference
	Gw *v1alpha1.Gateway
}

// markGatewayNodePhase marks the node with a phase, returns the node
func (gc *GatewayConfig) markGatewayNodePhase(nodeStatus *EventSourceStatus) *v1alpha1.NodeStatus {
	gc.Log.Debug().Str("node-name", nodeStatus.Name).Str("node-id", nodeStatus.Id).Str("phase", string(nodeStatus.Phase)).Msg("marking node phase")
	node := gc.getNodeByID(nodeStatus.Id)
	if node == nil {
		gc.Log.Warn().Str("node-name", nodeStatus.Name).Str("node-id", nodeStatus.Id).Msg("node is not initialized")
		return nil
	}
	if node.Phase != nodeStatus.Phase {
		gc.Log.Info().Str("node-name", nodeStatus.Name).Str("node-id", nodeStatus.Id).Str("old-phase", string(node.Phase)).Str("new-phase", string(nodeStatus.Phase)).Msg("phase marked")
		node.Phase = nodeStatus.Phase
	}
	node.Message = nodeStatus.Message
	gc.gw.Status.Nodes[node.ID] = *node
	gc.updated = true
	return node
}

// getNodeByName returns the node from this gateway for the nodeName
func (gc *GatewayConfig) getNodeByID(nodeID string) *v1alpha1.NodeStatus {
	node, ok := gc.gw.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

// create a new node
func (gc *GatewayConfig) initializeNode(nodeID string, nodeName string, messages string) v1alpha1.NodeStatus {
	if gc.gw.Status.Nodes == nil {
		gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	}
	gc.Log.Info().Str("node-id", nodeID).Str("node-name", nodeName).Msg("node")
	node, ok := gc.gw.Status.Nodes[nodeID]
	if !ok {
		node = v1alpha1.NodeStatus{
			ID:          nodeID,
			Name:        nodeName,
			DisplayName: nodeName,
			StartedAt:   metav1.MicroTime{Time: time.Now().UTC()},
		}
	}
	node.Phase = v1alpha1.NodePhaseRunning
	node.Message = messages
	gc.gw.Status.Nodes[nodeID] = node
	gc.Log.Info().Str("node-name", node.DisplayName).Str("node-message", node.Message).Msg("node is running")
	gc.updated = true
	return node
}

// UpdateGatewayResourceState updates gateway resource nodes state
func (gc *GatewayConfig) UpdateGatewayResourceState(status *EventSourceStatus) {
	gc.Log.Info().Msg("received a gateway state update notification")

	msg := fmt.Sprintf("event source state changed to %s", string(status.Phase))

	switch status.Phase {
	case v1alpha1.NodePhaseRunning:
		// init the node and mark it as running
		gc.initializeNode(status.Id, status.Name, status.Message)

	case v1alpha1.NodePhaseCompleted, v1alpha1.NodePhaseError:
		gc.markGatewayNodePhase(status)

	case v1alpha1.NodePhaseResourceUpdate:
		if gc.gw.Spec.Watchers != status.Gw.Spec.Watchers {
			msg = "gateway watchers updated"
		}

	case v1alpha1.NodePhaseRemove:
		delete(gc.gw.Status.Nodes, status.Id)
	}

	// persist changes and create K8s event logging the change
	eventType := common.StateChangeEventType
	labels := map[string]string{
		common.LabelGatewayEventSourceName: status.Name,
		common.LabelGatewayName:            gc.Name,
		common.LabelGatewayEventSourceID:   status.Id,
		common.LabelOperation:              "persist_event_source_state",
	}
	updatedGw, err := gtw.PersistUpdates(gc.gwcs, gc.gw, &gc.Log)
	if err != nil {
		gc.Log.Error().Err(err).Msg("failed to persist gateway resource updates, reverting to old state")
		eventType = common.EscalationEventType
	}

	// update gateway ref. in case of failure to persist updates, this is a deep copy of old gateway resource
	gc.gw = updatedGw
	labels[common.LabelEventType] = string(eventType)

	// generate a K8s event for persist event source state change
	if err := common.GenerateK8sEvent(gc.Clientset, msg, eventType, "event source state update", gc.Name, gc.Namespace, gc.controllerInstanceID, gateway.Kind, labels); err != nil {
		gc.Log.Error().Err(err).Str("event-source-name", status.Name).Msg("failed to create K8s event to log event source state change")
	}
}
