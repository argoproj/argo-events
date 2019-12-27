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
func (gatewayContext *GatewayContext) markNodePhase(notification *eventSourceUpdate) *v1alpha1.NodeStatus {
	logger := gatewayContext.logger.WithFields(
		map[string]interface{}{
			common.LabelNodeName: notification.name,
			common.LabelPhase:    string(notification.phase),
		},
	)

	logger.Infoln("marking node phase")

	node := gatewayContext.getNodeByID(notification.id)
	if node == nil {
		logger.Warnln("node is not initialized")
		return nil
	}
	if node.Phase != notification.phase {
		logger.WithField("new-phase", string(notification.phase)).Infoln("phase updated")
		node.Phase = notification.phase
	}
	node.Message = notification.message
	gatewayContext.gateway.Status.Nodes[node.ID] = *node
	gatewayContext.updated = true
	return node
}

// getNodeByName returns a gateway node by the id
func (gatewayContext *GatewayContext) getNodeByID(nodeID string) *v1alpha1.NodeStatus {
	node, ok := gatewayContext.gateway.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

// create a new node
func (gatewayContext *GatewayContext) initializeNode(nodeID string, nodeName string, messages string) v1alpha1.NodeStatus {
	if gatewayContext.gateway.Status.Nodes == nil {
		gatewayContext.gateway.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	}

	gatewayContext.logger.WithField(common.LabelNodeName, nodeName).Infoln("node")

	node, ok := gatewayContext.gateway.Status.Nodes[nodeID]
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
	gatewayContext.gateway.Status.Nodes[nodeID] = node

	gatewayContext.logger.WithFields(
		map[string]interface{}{
			common.LabelNodeName: nodeName,
			"node-message":       node.Message,
		},
	).Infoln("node is running")

	gatewayContext.updated = true
	return node
}

// UpdateGatewayState updates the gateway resource or the event source node status
func (gatewayContext *GatewayContext) UpdateGatewayState(notification *notification) {
	logger := gatewayContext.logger

	if notification.gatewayNotification != nil {
		logger.Infoln("received a gateway resource update notification")
		gatewayContext.gateway = notification.gatewayNotification.gateway
		logger.Infoln("checking if any new subscribers are added")
		gatewayContext.updateSubscriberClients()
	}

	if notification.eventSourceNotification != nil {
		logger = logger.WithField(common.LabelEventSource, notification.eventSourceNotification).Logger
		logger.Infoln("received a event source state update notification")

		switch notification.eventSourceNotification.phase {
		case v1alpha1.NodePhaseRunning:
			// init the node and mark it as running
			gatewayContext.initializeNode(notification.eventSourceNotification.id, notification.eventSourceNotification.name, notification.eventSourceNotification.message)

		case v1alpha1.NodePhaseCompleted, v1alpha1.NodePhaseError:
			gatewayContext.markNodePhase(notification.eventSourceNotification)

		case v1alpha1.NodePhaseRemove:
			delete(gatewayContext.gateway.Status.Nodes, notification.eventSourceNotification.id)
			logger.Infoln("event source is removed")
			gatewayContext.updated = true
		}
	}

	if gatewayContext.updated {
		oldGateway := gatewayContext.gateway.DeepCopy()
		updatedGw, err := gtw.PersistUpdates(gatewayContext.gatewayClient, gatewayContext.gateway, gatewayContext.logger)
		if err != nil {
			logger.WithError(err).Errorln("failed to persist gateway resource updates, reverting to old state")
			// in case of failure to persist updates, this is a deep copy of old gateway resource
			gatewayContext.gateway = oldGateway
		} else {
			gatewayContext.gateway = updatedGw
		}
	}
	gatewayContext.updated = false
}
