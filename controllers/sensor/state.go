package sensor

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"time"

	"github.com/rs/zerolog"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
	sclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
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

// GeneratePersistUpdateEvent persists sensor updates and generate a K8s event
func GeneratePersistUpdateEvent(k8client kubernetes.Interface, sensorclient sclient.Interface, sensor *v1alpha1.Sensor, controllerInstanceId string, log *zerolog.Logger) {
	labels := map[string]string{
		common.LabelSensorName:                    sensor.Name,
		common.LabelSensorKeyPhase:                string(sensor.Status.Phase),
		common.LabelKeySensorControllerInstanceID: controllerInstanceId,
		common.LabelOperation:                     "persist_state_update",
	}
	if err := PersistUpdates(sensorclient, sensor, controllerInstanceId, log); err != nil {
		log.Error().Err(err).Msg("failed to persist sensor update, escalating...")

		// escalate
		labels[common.LabelEventType] = string(common.EscalationEventType)
		if err := common.GenerateK8sEvent(k8client, "persist update", common.StateChangeEventType, "sensor state update", sensor.Name,
			sensor.Namespace, controllerInstanceId, sensor.Kind, labels); err != nil {
			log.Error().Err(err).Msg("failed to create K8s event to log sensor persist operation failure")
			return
		}
	}

	labels[common.LabelEventType] = string(common.StateChangeEventType)
	if err := common.GenerateK8sEvent(k8client, "persist update", common.StateChangeEventType, "sensor state update", sensor.Name, sensor.Namespace, controllerInstanceId, sensor.Kind, labels); err != nil {
		log.Error().Err(err).Msg("failed to create K8s event to log sensor state persist operation")
		return
	}

	log.Info().Msg("successfully persisted sensor resource update and created K8s event")
}

// PersistUpdates persists the updates to the Sensor resource
func PersistUpdates(client sclient.Interface, sensor *v1alpha1.Sensor, controllerInstanceId string, log *zerolog.Logger) error {
	var err error
	sensor, err = client.ArgoprojV1alpha1().Sensors(sensor.Namespace).Update(sensor)
	if err != nil {
		log.Warn().Err(err).Msg("error updating sensor")
		if errors.IsConflict(err) {
			return err
		}
		log.Info().Msg("re-applying updates on latest version and retrying update")
		err = ReapplyUpdate(client, sensor)
		if err != nil {
			log.Error().Err(err).Msg("failed to re-apply update")
			return err
		}
	}
	log.Info().Msg("sensor updated successfully")
	time.Sleep(1 * time.Second)
	return nil
}

// Reapply the update to sensor
func ReapplyUpdate(sensorClient sclient.Interface, sensor *v1alpha1.Sensor) error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		client := sensorClient.ArgoprojV1alpha1().Sensors(sensor.Namespace)
		s, err := client.Get(sensor.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		s.Status = sensor.Status
		sensor, err = client.Update(s)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
}

// MarkNodePhase marks the node with a phase, returns the node
func MarkNodePhase(sensor *v1alpha1.Sensor, nodeName string, nodeType v1alpha1.NodeType, phase v1alpha1.NodePhase, event *v1alpha1.Event, log *zerolog.Logger, message ...string) *v1alpha1.NodeStatus {
	node := GetNodeByName(sensor, nodeName)
	if node == nil {
		log.Panic().Str("node-name", nodeName).Msg("node is uninitialized")
	}
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
