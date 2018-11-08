package gateways

import (
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/api/errors"
	"github.com/argoproj/argo-events/common"
	"fmt"
)

// markGatewayNodePhase marks the node with a phase, returns the node
func (gc *GatewayConfig) markGatewayNodePhase(nodeID string, phase v1alpha1.NodePhase, message string) *v1alpha1.NodeStatus {
	gc.Log.Debug().Str("node-id", nodeID).Str("phase", string(phase)).Msg("marking node phase")
	gc.Log.Info().Interface("active-nodes", gc.gw.Status.Nodes).Msg("nodes")
	node := gc.getNodeByID(nodeID)
	if node == nil {
		gc.Log.Warn().Str("node-id", nodeID).Msg("node is not initialized")
		return nil
	}
	if node.Phase != v1alpha1.NodePhaseCompleted && node.Phase != phase {
		node.Phase = phase
		gc.Log.Info().Str("node-id", nodeID).Str("old-phase", string(node.Phase)).Str("new-phase", string(phase)).Msg("phase marked")
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
func (gc *GatewayConfig) initializeNode(nodeID string, nodeName string, timeID string, messages string) v1alpha1.NodeStatus {
	if gc.gw.Status.Nodes == nil {
		gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	}
	gc.Log.Info().Str("node-id", nodeID).Str("node-name", nodeName).Str("time-id", timeID).Msg("node")
	oldNode, ok := gc.gw.Status.Nodes[nodeID]
	if ok {
		gc.Log.Info().Str("node-name", nodeName).Msg("node already initialized")
		return oldNode
	}

	node := v1alpha1.NodeStatus{
		ID:          nodeID,
		TimeID:      timeID,
		Name:        nodeName,
		DisplayName: nodeName,
		Phase:       v1alpha1.NodePhaseInitialized,
		StartedAt:   metav1.MicroTime{Time: time.Now().UTC()},
	}
	node.Message = messages
	gc.gw.Status.Nodes[nodeID] = node
	gc.Log.Info().Str("node-name", node.DisplayName).Str("node-message", node.Message).Msg("node is initialized")
	return node
}

// updateGatewayResource updates gateway resource
func (gc *GatewayConfig) updateGatewayResource(event *corev1.Event) error {
	var err error

	defer func() {
		gc.Log.Info().Str("event-name", event.Name).Str("action", event.Action).Msg("marking gateway k8 event as seen")
		// mark event as seen
		event.ObjectMeta.Labels[common.LabelEventSeen] = "true"
		_, err = gc.Clientset.CoreV1().Events(gc.gw.Namespace).Update(event)
		if err != nil {
			gc.Log.Error().Err(err).Str("event-name", event.ObjectMeta.Name).Msg("failed to mark event as seen")
		}
	}()

	// its better to get latest resource version in case user performed an gateway resource update using kubectl
	gw, err := gc.gwcs.ArgoprojV1alpha1().Gateways(gc.gw.Namespace).Get(gc.gw.Name, metav1.GetOptions{})
	if err != nil {
		gc.Log.Error().Err(err).Str("event-name", event.Name).Msg("failed to retrieve the gateway")
		return err
	}

	gc.gw = gw

	// get time id of configuration
	timeID, ok := event.ObjectMeta.Labels[common.LabelGatewayConfigTimeID]
	if !ok {
		return fmt.Errorf("failed to apply update to gateway configuration. time ID is not present. event ID: %s", event.ObjectMeta.Name)
	}

	// get node/configuration to update
	nodeID, ok := event.ObjectMeta.Labels[common.LabelGatewayConfigID]
	if !ok {
		return fmt.Errorf("failed to update gateway resource. no configuration name provided")
	}
	node, ok := gc.gw.Status.Nodes[nodeID]
	gc.Log.Info().Str("config-id", nodeID).Str("action", string(event.Action)).Msg("k8 event")

	// initialize the configuration
	if !ok && v1alpha1.NodePhase(event.Action) == v1alpha1.NodePhaseInitialized {
		nodeName, ok := event.ObjectMeta.Labels[common.LabelGatewayConfigurationName]
		if !ok {
			gc.Log.Warn().Str("config-id", nodeID).Msg("configuration name is not provided")
			nodeName = nodeID
		}

		gc.Log.Warn().Str("config-id", nodeID).Msg("configuration not registered with gateway resource. initializing configuration...")
		gc.initializeNode(nodeID, nodeName, timeID, "initialized")
		return gc.persistUpdates()
	}

	gc.Log.Info().Str("config-name", nodeID).Msg("updating gateway resource...")

	// check if the event is actually valid and just arrived out of order
	if ok {
		if node.TimeID == timeID {
			// precedence of states Remove > Complete/Error > Running > Initialized
			switch v1alpha1.NodePhase(event.Action) {
			case v1alpha1.NodePhaseRunning, v1alpha1.NodePhaseCompleted, v1alpha1.NodePhaseError:
				gc.gw.Status.Nodes[nodeID] = node
				if node.Phase != v1alpha1.NodePhaseCompleted || node.Phase == v1alpha1.NodePhaseError {
					gc.markGatewayNodePhase(nodeID, v1alpha1.NodePhase(event.Action), event.Reason)
				}
				return gc.persistUpdates()
			case v1alpha1.NodePhaseRemove:
				gc.Log.Info().Str("config-name", nodeID).Msg("removing configuration from gateway")
				delete(gc.gw.Status.Nodes, nodeID)
				return gc.persistUpdates()
			default:
				gc.Log.Error().Str("config-name", nodeID).Str("event-name", event.Name).Str("event-action", event.Action).Msg("unknown action for configuration")
				return nil
			}
		} else {
			gc.Log.Error().Str("config-name", nodeID).Str("event-name", event.Name).Str("event-action", event.Action).
				Str("node-time-id", node.TimeID).Str("event-time-id", timeID).Msg("time ids mismatch")
			return nil
		}
	} else {
		gc.Log.Warn().Str("config-name", nodeID).Str("event-name", event.Name).Msg("skipping event")
		return nil
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
