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

package main

import (
	"time"

	"github.com/argoproj/argo-events/common"
	gtw "github.com/argoproj/argo-events/controllers/gateway"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// notification encapsulates state of an event source
type notification struct {
	// event source update notification
	eventSourceNotification *eventSourceUpdate
	// gateway resource update notification
	gatewayNotification *resourceUpdate
}

// eventSourceUpdate refers to update related to a event source
type eventSourceUpdate struct {
	// id of the event source
	id string
	// name of the event source
	name string
	// phase of the event source node within gateway resource
	phase v1alpha1.NodePhase
	// message refers to cause of phase change
	message string
}

// resourceUpdate refers to update related to gateway resource
type resourceUpdate struct {
	// gateway refers to updated gateway resource
	gateway *v1alpha1.Gateway
}

// markNodePhase marks the gateway node with a phase
func (gc *GatewayContext) markNodePhase(notification *eventSourceUpdate) *v1alpha1.NodeStatus {
	logger := gc.logger.WithFields(
		map[string]interface{}{
			common.LabelNodeName: notification.name,
			common.LabelPhase:    string(notification.phase),
		},
	)

	logger.Infoln("marking node phase")

	node := gc.getNodeByID(notification.id)
	if node == nil {
		logger.Warnln("node is not initialized")
		return nil
	}
	if node.Phase != notification.phase {
		logger.WithField("new-phase", string(notification.phase)).Infoln("phase updated")
		node.Phase = notification.phase
	}
	node.Message = notification.message
	gc.gateway.Status.Nodes[node.ID] = *node
	gc.updated = true
	return node
}

// getNodeByName returns a gateway node by the id
func (gc *GatewayContext) getNodeByID(nodeID string) *v1alpha1.NodeStatus {
	node, ok := gc.gateway.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

// create a new node
func (gc *GatewayContext) initializeNode(nodeID string, nodeName string, messages string) v1alpha1.NodeStatus {
	if gc.gateway.Status.Nodes == nil {
		gc.gateway.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	}

	gc.logger.WithField(common.LabelNodeName, nodeName).Infoln("initializing the node")

	node, ok := gc.gateway.Status.Nodes[nodeID]
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
	gc.gateway.Status.Nodes[nodeID] = node

	gc.logger.WithFields(
		map[string]interface{}{
			common.LabelNodeName: nodeName,
			"node-message":       node.Message,
		},
	).Infoln("node is running")

	gc.updated = true
	return node
}

// UpdateGatewayState updates the gateway resource or the event source node status
func (gc *GatewayContext) UpdateGatewayState(notification *notification) {
	defer func() {
		gc.updated = false
	}()

	logger := gc.logger

	if notification.gatewayNotification != nil {
		logger.Infoln("received a gateway resource update notification")
		gc.gateway = notification.gatewayNotification.gateway
		logger.Infoln("checking if any new subscribers are added")
		gc.updateSubscriberClients()
	}

	if notification.eventSourceNotification != nil {
		logger = logger.WithField(common.LabelEventSource, notification.eventSourceNotification).Logger
		logger.Infoln("received a event source state update notification")

		switch notification.eventSourceNotification.phase {
		case v1alpha1.NodePhaseRunning:
			// init the node and mark it as running
			gc.initializeNode(notification.eventSourceNotification.id, notification.eventSourceNotification.name, notification.eventSourceNotification.message)

		case v1alpha1.NodePhaseCompleted, v1alpha1.NodePhaseError:
			gc.markNodePhase(notification.eventSourceNotification)

		case v1alpha1.NodePhaseRemove:
			delete(gc.gateway.Status.Nodes, notification.eventSourceNotification.id)
			logger.Infoln("event source is removed")
			gc.updated = true
		}
	}

	if gc.updated {
		updatedGw, err := gtw.PersistUpdates(gc.gatewayClient, gc.gateway, gc.logger)
		if err != nil {
			logger.WithError(err).Errorln("failed to persist gateway resource updates, reverting to old state")
			return
		}
		gc.gateway = updatedGw
	}
}
