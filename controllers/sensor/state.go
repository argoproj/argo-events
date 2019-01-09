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
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"time"

	"github.com/argoproj/argo-events/common"
	sclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// create a new node
func InitializeNode(sensor *v1alpha1.Sensor, nodeName string, nodeType v1alpha1.NodeType, phase v1alpha1.NodePhase, log *zerolog.Logger, messages ...string) *v1alpha1.NodeStatus {
	if sensor.Status.Nodes == nil {
		sensor.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	}
	nodeID := sensor.NodeID(nodeName)
	oldNode, ok := sensor.Status.Nodes[nodeID]
	if ok {
		log.Info().Str("node-name", nodeName).Msg("node already initialized")
		return &oldNode
	}
	node := v1alpha1.NodeStatus{
		ID:          nodeID,
		Name:        nodeName,
		DisplayName: nodeName,
		Type:        nodeType,
		Phase:       phase,
		StartedAt:   metav1.MicroTime{Time: time.Now().UTC()},
	}
	if len(messages) > 0 {
		node.Message = messages[0]
	}
	sensor.Status.Nodes[nodeID] = node
	log.Info().Str("node-type", string(node.Type)).Str("node-name", node.DisplayName).Str("node-message", node.Message).Msg("node is initialized")
	return &node
}

// PersistUpdates persists the updates to the Sensor resource
func PersistUpdates(client sclient.Interface, sensor *v1alpha1.Sensor, controllerInstanceId string, log *zerolog.Logger) (*v1alpha1.Sensor, error) {
	sensorClient := client.ArgoprojV1alpha1().Sensors(sensor.ObjectMeta.Namespace)
	// in case persist update fails
	oldsensor := sensor.DeepCopy()

	sensor, err := sensorClient.Update(sensor)
	if err != nil {
		log.Warn().Err(err).Msg("error updating sensor")
		if errors.IsConflict(err) {
			return oldsensor, err
		}
		log.Info().Msg("re-applying updates on latest version and retrying update")
		err = ReapplyUpdate(client, sensor)
		if err != nil {
			log.Error().Err(err).Msg("failed to re-apply update")
			return oldsensor, err
		}
	}
	log.Info().Str("phase", string(sensor.Status.Phase)).Msg("sensor state updated successfully")
	return sensor, nil
}

// Reapply the update to sensor
func ReapplyUpdate(sensorClient sclient.Interface, sensor *v1alpha1.Sensor) error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		client := sensorClient.ArgoprojV1alpha1().Sensors(sensor.Namespace)
		s, err := client.Update(sensor)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		sensor = s
		return true, nil
	})
}

// MarkNodePhase marks the node with a phase, returns the node
func MarkNodePhase(sensor *v1alpha1.Sensor, nodeName string, nodeType v1alpha1.NodeType, phase v1alpha1.NodePhase, event *v1alpha1.Event, log *zerolog.Logger, message ...string) *v1alpha1.NodeStatus {
	node := GetNodeByName(sensor, nodeName)
	if node.Phase != phase {
		log.Info().Str("type", string(node.Type)).Str("node-name", node.Name).Str("phase", string(node.Phase))
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
		log.Info().Str("type", string(node.Type)).Str("node-name", node.Name).Msg("completed")
	}

	sensor.Status.Nodes[node.ID] = *node
	return node
}
