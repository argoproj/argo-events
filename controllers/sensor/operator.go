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
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorclientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// the context of an operation on a sensor.
// the controller creates this context each time it picks a Sensor off its queue.
type operationContext struct {
	// sensor is the sensor object
	sensor *v1alpha1.Sensor
	// updated indicates whether the sensor object was updated and needs to be persisted back to k8
	updated bool
	// logger logs stuff
	logger *logrus.Logger
	// reference to the controller
	controller *Controller
}

// newOperationCtx creates and initializes a new operationContext object
func newOperationCtx(sensorObj *v1alpha1.Sensor, controller *Controller) *operationContext {
	return &operationContext{
		sensor:  sensorObj.DeepCopy(),
		updated: false,
		logger: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelSensorName: sensorObj.Name,
				common.LabelNamespace:  sensorObj.Namespace,
			}).Logger,
		controller: controller,
	}
}

// operate manages the lifecycle of a sensor object
func (opctx *operationContext) operate() error {
	defer opctx.updateSensorState()

	// Validation failure prevents any sort processing of the sensor object
	if err := ValidateSensor(opctx.sensor); err != nil {
		opctx.logger.WithError(err).Errorln("failed to validate sensor")
		opctx.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		return err
	}

	switch opctx.sensor.Status.Phase {
	case v1alpha1.NodePhaseNew:
		// If the sensor phase is new
		// 1. Initialize all nodes - dependencies, dependency groups and triggers
		// 2. Make dependencies and dependency groups as active
		// 3. Create a deployment and service (if needed) for the sensor
		opctx.logger.Infoln("initializing the sensor object...")
		opctx.initializeAllNodes()
		opctx.markDependencyNodesActive()

		err := opctx.createResources()
		if err != nil {
			return err
		}
		opctx.logger.Infoln("successfully processed sensor state update")

	case v1alpha1.NodePhaseActive:
		// If the sensor is already active and if the sensor podTemplate spec has changed, then update the corresponding deployment
		opctx.logger.Infoln("sensor is already running, checking for updates to the sensor object...")
		err := opctx.updateResources()
		if err != nil {
			return err
		}
		opctx.logger.Infoln("successfully processed sensor state update")

	case v1alpha1.NodePhaseError:
		// If the sensor is in error state and if the sensor podTemplate spec has changed, then update the corresponding deployment
		opctx.logger.Info("sensor is in error state, checking for updates to the sensor object...")
		err := opctx.updateResources()
		if err != nil {
			return err
		}
		opctx.logger.Infoln("successfully processed sensor state update")
	}

	return nil
}

// updateSensorState updates the sensor resource state
func (opctx *operationContext) updateSensorState() {
	if opctx.updated {
		// persist updates to sensor resource
		labels := map[string]string{
			common.LabelSensorName:    opctx.sensor.Name,
			LabelPhase:                string(opctx.sensor.Status.Phase),
			LabelControllerInstanceID: opctx.controller.Config.InstanceID,
			common.LabelOperation:     "persist_state_update",
		}
		eventType := common.StateChangeEventType

		updatedSensor, err := PersistUpdates(opctx.controller.sensorClient, opctx.sensor, opctx.logger)
		if err != nil {
			opctx.logger.WithError(err).Errorln("failed to persist sensor update")

			// escalate failure
			eventType = common.EscalationEventType
		}

		// update sensor ref. in case of failure to persist updates, this is a deep copy of old sensor resource
		opctx.sensor = updatedSensor

		labels[common.LabelEventType] = string(eventType)
		if err := common.GenerateK8sEvent(opctx.controller.k8sClient,
			"persist update",
			eventType,
			"sensor state update",
			opctx.sensor.Name,
			opctx.sensor.Namespace,
			opctx.controller.Config.InstanceID,
			sensor.Kind,
			labels); err != nil {
			opctx.logger.WithError(err).Error("failed to create K8s event to logger sensor state persist operation")
			return
		}
		opctx.logger.Info("successfully persisted sensor resource update and created K8s event")
	}
	opctx.updated = false
}

// mark the overall sensor phase
func (opctx *operationContext) markSensorPhase(phase v1alpha1.NodePhase, markComplete bool, message ...string) {
	justCompleted := opctx.sensor.Status.Phase != phase
	if justCompleted {
		opctx.logger.WithFields(
			map[string]interface{}{
				"old": string(opctx.sensor.Status.Phase),
				"new": string(phase),
			},
		).Infoln("phase updated")

		opctx.sensor.Status.Phase = phase

		if opctx.sensor.ObjectMeta.Labels == nil {
			opctx.sensor.ObjectMeta.Labels = make(map[string]string)
		}

		if opctx.sensor.ObjectMeta.Annotations == nil {
			opctx.sensor.ObjectMeta.Annotations = make(map[string]string)
		}

		opctx.sensor.ObjectMeta.Labels[LabelPhase] = string(phase)
		opctx.sensor.ObjectMeta.Annotations[LabelPhase] = string(phase)
	}

	if opctx.sensor.Status.StartedAt.IsZero() {
		opctx.sensor.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}

	if len(message) > 0 && opctx.sensor.Status.Message != message[0] {
		opctx.logger.WithFields(
			map[string]interface{}{
				"old": opctx.sensor.Status.Message,
				"new": message[0],
			},
		).Infoln("sensor message updated")

		opctx.sensor.Status.Message = message[0]
	}

	switch phase {
	case v1alpha1.NodePhaseError:
		if markComplete && justCompleted {
			opctx.logger.Infoln("marking sensor state as complete")
			opctx.sensor.Status.CompletedAt = metav1.Time{Time: time.Now().UTC()}

			if opctx.sensor.ObjectMeta.Labels == nil {
				opctx.sensor.ObjectMeta.Labels = make(map[string]string)
			}
			if opctx.sensor.ObjectMeta.Annotations == nil {
				opctx.sensor.ObjectMeta.Annotations = make(map[string]string)
			}

			opctx.sensor.ObjectMeta.Labels[LabelComplete] = "true"
			opctx.sensor.ObjectMeta.Annotations[LabelComplete] = string(phase)
		}
	}
	opctx.updated = true
}

// initializeAllNodes initializes nodes of all types within a sensor
func (opctx *operationContext) initializeAllNodes() {
	// Initialize all event dependency nodes
	for _, dependency := range opctx.sensor.Spec.Dependencies {
		InitializeNode(opctx.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, opctx.logger)
	}

	// Initialize all dependency groups
	if opctx.sensor.Spec.DependencyGroups != nil {
		for _, group := range opctx.sensor.Spec.DependencyGroups {
			InitializeNode(opctx.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, opctx.logger)
		}
	}

	// Initialize all trigger nodes
	for _, trigger := range opctx.sensor.Spec.Triggers {
		InitializeNode(opctx.sensor, trigger.Template.Name, v1alpha1.NodeTypeTrigger, opctx.logger)
	}
}

// markDependencyNodesActive marks phase of all dependencies and dependency groups as active
func (opctx *operationContext) markDependencyNodesActive() {
	// Mark all event dependency nodes as active
	for _, dependency := range opctx.sensor.Spec.Dependencies {
		MarkNodePhase(opctx.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, opctx.logger, "node is active")
	}

	// Mark all dependency groups as active
	if opctx.sensor.Spec.DependencyGroups != nil {
		for _, group := range opctx.sensor.Spec.DependencyGroups {
			MarkNodePhase(opctx.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, opctx.logger, "node is active")
		}
	}
}

// PersistUpdates persists the updates to the Sensor resource
func PersistUpdates(client sensorclientset.Interface, sensorObj *v1alpha1.Sensor, log *logrus.Logger) (*v1alpha1.Sensor, error) {
	sensorClient := client.ArgoprojV1alpha1().Sensors(sensorObj.ObjectMeta.Namespace)
	// in case persist update fails
	oldsensor := sensorObj.DeepCopy()

	sensorObj, err := sensorClient.Update(sensorObj)
	if err != nil {
		if errors.IsConflict(err) {
			log.WithError(err).Error("error updating sensor")
			return oldsensor, err
		}

		log.Info("re-applying updates on latest version and retrying update")
		err = ReapplyUpdate(client, sensorObj)
		if err != nil {
			log.WithError(err).Error("failed to re-apply update")
			return oldsensor, err
		}
	}
	log.WithField(common.LabelPhase, string(sensorObj.Status.Phase)).Info("sensor state updated successfully")
	return sensorObj, nil
}

// Reapply the update to sensor
func ReapplyUpdate(sensorClient sensorclientset.Interface, sensor *v1alpha1.Sensor) error {
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
