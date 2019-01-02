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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

// EventSourceStatus encapsulates state of an event source
type EventSourceStatus struct {
	Id string
	Name string
	Message string
	Phase v1alpha1.NodePhase
}

// markGatewayNodePhase marks the node with a phase, returns the node
func (gc *GatewayConfig) markGatewayNodePhase(nodeStatus *EventSourceStatus) *v1alpha1.NodeStatus {
	gc.Log.Debug().Str("node-name", nodeStatus.Name).Str("node-id", nodeStatus.Id).Str("phase", string(nodeStatus.Phase)).Msg("marking node phase")
	node := gc.getNodeByID(nodeStatus.Id)
	if node == nil {
		gc.Log.Warn().Str("node-name", nodeStatus.Name).Str("node-id", nodeStatus.Id).Msg("node is not initialized")
		return nil
	}
	if node.Phase != v1alpha1.NodePhaseCompleted && node.Phase != phase {
		gc.Log.Info().Str("node-id", nodeID).Str("old-phase", string(node.Phase)).Str("new-phase", string(phase)).Msg("phase marked")
		node.Phase = phase
	}
	node.Message = message
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

// updateGatewayResourceState updates gateway resource nodes state
func (gc *GatewayConfig) updateGatewayResourceState(status *EventSourceStatus) {
	defer func() {
		err := gc.persistUpdates()
		if err != nil {
			fmt.Println("failed to persist gateway resource updates", err)
		}
	}()

	switch status.Phase {
	case v1alpha1.NodePhaseRunning:
		// create a node and mark it as running
		gc.initializeNode(status.Id, status.Name, status.Message)
	case v1alpha1.NodePhaseCompleted, v1alpha1.NodePhaseError:
		gc.markGatewayNodePhase(status)
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
