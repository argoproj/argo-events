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
	"time"

	"github.com/argoproj/argo-events/pkg/apis/gateway"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
	oldNode, ok := gc.gw.Status.Nodes[nodeID]
	if ok {
		gc.Log.Info().Str("node-name", nodeName).Msg("node already initialized")
		return oldNode
	}

	node := v1alpha1.NodeStatus{
		ID:          nodeID,
		Name:        nodeName,
		DisplayName: nodeName,
		Phase:       v1alpha1.NodePhaseRunning,
		StartedAt:   metav1.MicroTime{Time: time.Now().UTC()},
	}
	node.Message = messages
	gc.gw.Status.Nodes[nodeID] = node
	gc.Log.Info().Str("node-name", node.DisplayName).Str("node-message", node.Message).Msg("node is running")
	return node
}

// UpdateGatewayEventSourceState updates gateway resource nodes state
func (gc *GatewayConfig) UpdateGatewayEventSourceState(status *EventSourceStatus) {
	var msg string

	// create a deep copy of old gateway resource, so in the event of persist update failure, we can revert back
	oldgw := gc.gw.DeepCopy()

	// persist changes and create K8s event logging the change
	defer func(msg *string) {
		err := gc.persistUpdates()
		if err != nil {
			fmt.Println("failed to persist gateway resource updates, reverting to old state", err)
			gc.gw = oldgw
			// escalate
			labels := map[string]string{
				common.LabelEventType:              string(common.EscalationEventType),
				common.LabelGatewayEventSourceName: status.Name,
				common.LabelGatewayName:            gc.Name,
				common.LabelGatewayEventSourceID:   status.Id,
				common.LabelOperation:              "failed to update event source state",
			}
			if err := common.GenerateK8sEvent(gc.Clientset, fmt.Sprintf("failed to update event source state. err: %+v", err), common.StateChangeEventType, "event source state update", gc.Name, gc.Namespace, gc.controllerInstanceID, gateway.Kind, labels); err != nil {
				gc.Log.Error().Err(err).Str("event-source-name", status.Name).Msg("failed to create K8s event to log event source state change failure")
			}
		}
		// generate a K8s event for event source state change
		labels := map[string]string{
			common.LabelEventType:              string(common.StateChangeEventType),
			common.LabelGatewayEventSourceName: status.Name,
			common.LabelGatewayName:            gc.Name,
			common.LabelGatewayEventSourceID:   status.Id,
			common.LabelOperation:              "update event source state",
		}
		if err := common.GenerateK8sEvent(gc.Clientset, *msg, common.StateChangeEventType, "event source state update", gc.Name, gc.Namespace, gc.controllerInstanceID, gateway.Kind, labels); err != nil {
			gc.Log.Error().Err(err).Str("event-source-name", status.Name).Msg("failed to create K8s event to log event source state change")
		}
	}(&msg)

	msg = fmt.Sprintf("event source state changed to %s", string(status.Phase))

	switch status.Phase {
	case v1alpha1.NodePhaseRunning:
		// create a node and mark it as running
		gc.initializeNode(status.Id, status.Name, status.Message)

	case v1alpha1.NodePhaseCompleted, v1alpha1.NodePhaseError:
		gc.markGatewayNodePhase(status)

	case v1alpha1.NodePhaseResourceUpdate:
		gc.gw.Spec.Watchers = status.Gw.Spec.Watchers
		gc.gw.Spec.EventVersion = status.Gw.Spec.EventVersion
		gc.gw.Spec.Type = status.Gw.Spec.Type
		msg = "gateway watchers and event metadata updated"

	case v1alpha1.NodePhaseRemove:
		delete(gc.gw.Status.Nodes, status.Id)
	}
}

// PersistUpdates persists the updates to the Gateway resource
func (gc *GatewayConfig) persistUpdates() error {
	var err error
	gc.gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gc.gw.Namespace).Update(gc.gw)
	if err != nil {
		gc.Log.Warn().Err(err).Msg("error updating gateway")
		if errors.IsConflict(err) {
			return err
		}
		gc.Log.Info().Msg("re-applying updates on latest version and retrying update")
		err = gc.reapplyUpdate()
		if err != nil {
			gc.Log.Error().Err(err).Msg("failed to re-apply update")
			return err
		}
	}
	gc.Log.Info().Msg("gateway updated successfully")
	time.Sleep(1 * time.Second)
	return nil
}

// reapplyUpdate by fetching a new version of the sensor and updating the status
func (gc *GatewayConfig) reapplyUpdate() error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		gwClient := gc.gwcs.ArgoprojV1alpha1().Gateways(gc.gw.Namespace)
		gw, err := gwClient.Get(gc.gw.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		gc.gw.Status = gw.Status
		gc.gw, err = gwClient.Update(gc.gw)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
}
