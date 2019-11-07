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
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// the context of an operation on a sensor.
// the controller creates this context each time it picks a Sensor off its queue.
type operationContext struct {
	// sensorObj is the sensor object
	sensorObj *v1alpha1.Sensor
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
		sensorObj: sensorObj.DeepCopy(),
		updated:   false,
		logger: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelSensorName: sensorObj.Name,
				common.LabelNamespace:  sensorObj.Namespace,
			}).Logger,
		controller: controller,
	}
}

// operate on the sensor resource
func (opctx *operationContext) operate() error {
	defer opctx.updateSensorState()

	if err := ValidateSensor(opctx.sensorObj); err != nil {
		opctx.logger.WithError(err).Error("failed to validate sensor")
		opctx.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		return err
	}

	switch opctx.sensorObj.Status.Phase {
	case v1alpha1.NodePhaseNew:
		opctx.logger.Infoln("initializing the sensor object...")

		opctx.initializeAllNodes()
		err := opctx.createSensorResources()
		if err != nil {
			return err
		}

	case v1alpha1.NodePhaseActive:
		opctx.logger.Info("sensor is already running, checking for updates to the sensor object...")

		err := opctx.updateSensorResources()
		if err != nil {
			return err
		}

	case v1alpha1.NodePhaseError:
		opctx.logger.Info("sensor is in error state, checking for updates to the sensor object...")

		err := opctx.updateSensorResources()
		if err != nil {
			return err
		}
	}
	return nil
}

// updateSensorState updates the sensor resource state
func (opctx *operationContext) updateSensorState() {
	if opctx.updated {
		// persist updates to sensor resource
		labels := map[string]string{
			common.LabelSensorName:                    opctx.sensorObj.Name,
			common.LabelSensorKeyPhase:                string(opctx.sensorObj.Status.Phase),
			common.LabelKeySensorControllerInstanceID: opctx.controller.Config.InstanceID,
			common.LabelOperation:                     "persist_state_update",
		}
		eventType := common.StateChangeEventType

		updatedSensor, err := PersistUpdates(opctx.controller.sensorClient, opctx.sensorObj, opctx.controller.Config.InstanceID, opctx.logger)
		if err != nil {
			opctx.logger.WithError(err).Error("failed to persist sensor update")

			// escalate failure
			eventType = common.EscalationEventType
		}

		// update sensor ref. in case of failure to persist updates, this is a deep copy of old sensor resource
		opctx.sensorObj = updatedSensor

		labels[common.LabelEventType] = string(eventType)
		if err := common.GenerateK8sEvent(opctx.controller.k8sClient,
			"persist update",
			eventType,
			"sensor state update",
			opctx.sensorObj.Name,
			opctx.sensorObj.Namespace,
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
	justCompleted := opctx.sensorObj.Status.Phase != phase
	if justCompleted {
		opctx.logger.WithFields(
			map[string]interface{}{
				"old": string(opctx.sensorObj.Status.Phase),
				"new": string(phase),
			},
		).Infoln("phase updated")

		opctx.sensorObj.Status.Phase = phase

		if opctx.sensorObj.ObjectMeta.Labels == nil {
			opctx.sensorObj.ObjectMeta.Labels = make(map[string]string)
		}

		if opctx.sensorObj.ObjectMeta.Annotations == nil {
			opctx.sensorObj.ObjectMeta.Annotations = make(map[string]string)
		}

		opctx.sensorObj.ObjectMeta.Labels[common.LabelSensorKeyPhase] = string(phase)
		// add annotations so a resource sensor can watch this sensor.
		opctx.sensorObj.ObjectMeta.Annotations[common.LabelSensorKeyPhase] = string(phase)
	}

	if opctx.sensorObj.Status.StartedAt.IsZero() {
		opctx.sensorObj.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}

	if len(message) > 0 && opctx.sensorObj.Status.Message != message[0] {
		opctx.logger.WithFields(
			map[string]interface{}{
				"old": opctx.sensorObj.Status.Message,
				"new": message[0],
			},
		).Infoln("sensor message updated")

		opctx.sensorObj.Status.Message = message[0]
	}

	switch phase {
	case v1alpha1.NodePhaseError:
		if markComplete && justCompleted {
			opctx.logger.Infoln("marking sensor state as complete")
			opctx.sensorObj.Status.CompletedAt = metav1.Time{Time: time.Now().UTC()}

			if opctx.sensorObj.ObjectMeta.Labels == nil {
				opctx.sensorObj.ObjectMeta.Labels = make(map[string]string)
			}
			if opctx.sensorObj.ObjectMeta.Annotations == nil {
				opctx.sensorObj.ObjectMeta.Annotations = make(map[string]string)
			}

			opctx.sensorObj.ObjectMeta.Labels[common.LabelSensorKeyComplete] = "true"
			opctx.sensorObj.ObjectMeta.Annotations[common.LabelSensorKeyComplete] = string(phase)
		}
	}
	opctx.updated = true
}

// initializeAllNodes initializes nodes of all types within a sensor
func (opctx *operationContext) initializeAllNodes() {
	// Initialize all event dependency nodes
	for _, dependency := range opctx.sensorObj.Spec.Dependencies {
		InitializeNode(opctx.sensorObj, dependency.Name, v1alpha1.NodeTypeEventDependency, opctx.logger)
	}

	// Initialize all dependency groups
	if opctx.sensorObj.Spec.DependencyGroups != nil {
		for _, group := range opctx.sensorObj.Spec.DependencyGroups {
			InitializeNode(opctx.sensorObj, group.Name, v1alpha1.NodeTypeDependencyGroup, opctx.logger)
		}
	}

	// Initialize all trigger nodes
	for _, trigger := range opctx.sensorObj.Spec.Triggers {
		InitializeNode(opctx.sensorObj, trigger.Template.Name, v1alpha1.NodeTypeTrigger, opctx.logger)
	}
}

// markDependencyNodesActive marks phase of all dependencies and dependency groups as active
func (opctx *operationContext) markDependencyNodesActive() {
	// Mark all event dependency nodes as active
	for _, dependency := range opctx.sensorObj.Spec.Dependencies {
		MarkNodePhase(opctx.sensorObj, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, opctx.logger, "node is active")
	}

	// Mark all dependency groups as active
	if opctx.sensorObj.Spec.DependencyGroups != nil {
		for _, group := range opctx.sensorObj.Spec.DependencyGroups {
			MarkNodePhase(opctx.sensorObj, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, opctx.logger, "node is active")
		}
	}
}
