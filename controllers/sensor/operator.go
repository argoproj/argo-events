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
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorclientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// the context of an operation on a sensor.
// the controller creates this context each time it picks a Sensor off its queue.
type sensorContext struct {
	// sensor is the sensor object
	sensor *v1alpha1.Sensor
	// updated indicates whether the sensor object was updated and needs to be persisted back to k8
	updated bool
	// logger logs stuff
	logger *logrus.Logger
	// reference to the controller
	controller *Controller
}

// newSensorContext creates and initializes a new sensorContext object
func newSensorContext(sensorObj *v1alpha1.Sensor, controller *Controller) *sensorContext {
	return &sensorContext{
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
func (ctx *sensorContext) operate() error {
	defer ctx.updateSensorState()

	ctx.logger.Infoln("processing the sensor")

	// Validation failure prevents any sort processing of the sensor object
	if err := ValidateSensor(ctx.sensor); err != nil {
		ctx.logger.WithError(err).Errorln("failed to validate sensor")
		ctx.markSensorPhase(v1alpha1.NodePhaseError, err.Error())
		return err
	}

	switch ctx.sensor.Status.Phase {
	case v1alpha1.NodePhaseNew:
		// If the sensor phase is new
		// 1. Initialize all nodes - dependencies, dependency groups and triggers
		// 2. Make dependencies and dependency groups as active
		// 3. Create a deployment and service (if needed) for the sensor
		ctx.initializeAllNodes()
		ctx.markDependencyNodesActive()

		if err := ctx.createSensorResources(); err != nil {
			ctx.logger.WithError(err).Errorln("failed to create resources for the sensor")
			ctx.markSensorPhase(v1alpha1.NodePhaseError, err.Error())
			return nil
		}
		ctx.markSensorPhase(v1alpha1.NodePhaseActive, "sensor is active")
		ctx.logger.Infoln("successfully created resources for the sensor. sensor is in active state")

	case v1alpha1.NodePhaseActive:
		ctx.logger.Infoln("checking for updates to the sensor object")
		if err := ctx.updateSensorResources(); err != nil {
			ctx.logger.WithError(err).Errorln("failed to update the sensor resources")
			return err
		}
		ctx.logger.Infoln("successfully processed sensor state update")

	case v1alpha1.NodePhaseError:
		// If the sensor is in error state and if the sensor podTemplate spec has changed, then update the corresponding deployment
		ctx.logger.Info("sensor is in error state, checking for updates to the sensor object")
		if err := ctx.updateSensorResources(); err != nil {
			ctx.logger.WithError(err).Errorln("failed to update the sensor resources")
			return err
		}
		ctx.markSensorPhase(v1alpha1.NodePhaseActive, "sensor is active")
		ctx.logger.Infoln("successfully processed the update")
	}

	return nil
}

// createSensorResources creates the K8s resources for a sensor object
func (ctx *sensorContext) createSensorResources() error {
	if ctx.sensor.Status.Resources == nil {
		ctx.sensor.Status.Resources = &v1alpha1.SensorResources{}
	}

	ctx.logger.Infoln("generating deployment specification for the sensor")
	deployment, err := ctx.deploymentBuilder()
	if err != nil {
		return err
	}
	ctx.logger.WithField("name", deployment.Name).Infoln("creating the deployment resource for the sensor")
	deployment, err = ctx.createDeployment(deployment)
	if err != nil {
		return err
	}
	ctx.sensor.Status.Resources.Deployment = &deployment.ObjectMeta

	ctx.logger.Infoln("generating service specification for the sensor")
	service, err := ctx.serviceBuilder()
	if err != nil {
		return err
	}

	ctx.logger.WithField("name", service.Name).Infoln("generating deployment specification for the sensor")
	service, err = ctx.createService(service)
	if err != nil {
		return err
	}
	ctx.sensor.Status.Resources.Service = &service.ObjectMeta
	return err
}

// updateSensorResources updates the sensor resources
func (ctx *sensorContext) updateSensorResources() error {
	deployment, err := ctx.updateDeployment()
	if err != nil {
		return err
	}
	ctx.sensor.Status.Resources.Deployment = &deployment.ObjectMeta
	service, err := ctx.updateService()
	if err != nil {
		return err
	}
	if service == nil {
		ctx.sensor.Status.Resources.Service = nil
		return nil
	}
	ctx.sensor.Status.Resources.Service = &service.ObjectMeta
	return nil
}

// updateSensorState updates the sensor resource state
func (ctx *sensorContext) updateSensorState() {
	defer func() {
		ctx.updated = false
	}()
	if ctx.updated {
		updatedSensor, err := PersistUpdates(ctx.controller.sensorClient, ctx.sensor, ctx.logger)
		if err != nil {
			ctx.logger.WithError(err).Errorln("failed to persist sensor update")
			return
		}
		ctx.sensor = updatedSensor
		ctx.logger.Info("successfully persisted sensor resource update")
	}
}

// mark the overall sensor phase
func (ctx *sensorContext) markSensorPhase(phase v1alpha1.NodePhase, message ...string) {
	justCompleted := ctx.sensor.Status.Phase != phase
	if justCompleted {
		ctx.logger.WithFields(
			map[string]interface{}{
				"old": string(ctx.sensor.Status.Phase),
				"new": string(phase),
			},
		).Infoln("phase updated")

		ctx.sensor.Status.Phase = phase

		if ctx.sensor.ObjectMeta.Labels == nil {
			ctx.sensor.ObjectMeta.Labels = make(map[string]string)
		}

		if ctx.sensor.ObjectMeta.Annotations == nil {
			ctx.sensor.ObjectMeta.Annotations = make(map[string]string)
		}

		ctx.sensor.ObjectMeta.Labels[LabelPhase] = string(phase)
		ctx.sensor.ObjectMeta.Annotations[LabelPhase] = string(phase)
	}

	if ctx.sensor.Status.StartedAt.IsZero() {
		ctx.sensor.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}

	if len(message) > 0 && ctx.sensor.Status.Message != message[0] {
		ctx.logger.WithFields(
			map[string]interface{}{
				"old": ctx.sensor.Status.Message,
				"new": message[0],
			},
		).Infoln("sensor message updated")

		ctx.sensor.Status.Message = message[0]
	}

	if phase == v1alpha1.NodePhaseError && justCompleted {
		ctx.logger.Infoln("marking sensor state as complete")
		ctx.sensor.Status.CompletedAt = metav1.Time{Time: time.Now().UTC()}

		if ctx.sensor.ObjectMeta.Labels == nil {
			ctx.sensor.ObjectMeta.Labels = make(map[string]string)
		}
		if ctx.sensor.ObjectMeta.Annotations == nil {
			ctx.sensor.ObjectMeta.Annotations = make(map[string]string)
		}

		ctx.sensor.ObjectMeta.Labels[LabelComplete] = "true"
		ctx.sensor.ObjectMeta.Annotations[LabelComplete] = string(phase)
	}
	ctx.updated = true
}

// initializeAllNodes initializes nodes of all types within a sensor
func (ctx *sensorContext) initializeAllNodes() {
	// Initialize all event dependency nodes
	for _, dependency := range ctx.sensor.Spec.Dependencies {
		InitializeNode(ctx.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, ctx.logger)
	}

	// Initialize all dependency groups
	if ctx.sensor.Spec.DependencyGroups != nil {
		for _, group := range ctx.sensor.Spec.DependencyGroups {
			InitializeNode(ctx.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, ctx.logger)
		}
	}

	// Initialize all trigger nodes
	for _, trigger := range ctx.sensor.Spec.Triggers {
		InitializeNode(ctx.sensor, trigger.Template.Name, v1alpha1.NodeTypeTrigger, ctx.logger)
	}
}

// markDependencyNodesActive marks phase of all dependencies and dependency groups as active
func (ctx *sensorContext) markDependencyNodesActive() {
	// Mark all event dependency nodes as active
	for _, dependency := range ctx.sensor.Spec.Dependencies {
		MarkNodePhase(ctx.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, ctx.logger, "node is active")
	}

	// Mark all dependency groups as active
	if ctx.sensor.Spec.DependencyGroups != nil {
		for _, group := range ctx.sensor.Spec.DependencyGroups {
			MarkNodePhase(ctx.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, ctx.logger, "node is active")
		}
	}
}

// PersistUpdates persists the updates to the Sensor resource
func PersistUpdates(client sensorclientset.Interface, sensorObj *v1alpha1.Sensor, log *logrus.Logger) (*v1alpha1.Sensor, error) {
	sensorClient := client.ArgoprojV1alpha1().Sensors(sensorObj.ObjectMeta.Namespace)

	updatedSensor, err := sensorClient.Update(sensorObj)
	if err != nil {
		if errors.IsConflict(err) {
			log.WithError(err).Error("error updating sensor")
			return nil, err
		}

		log.Infoln(err)
		log.Infoln("re-applying updates on latest version and retrying update")
		err = ReapplyUpdate(client, sensorObj)
		if err != nil {
			log.WithError(err).Error("failed to re-apply update")
			return nil, err
		}
		return sensorObj, nil
	}
	log.WithField(common.LabelPhase, string(sensorObj.Status.Phase)).Info("sensor state updated successfully")
	return updatedSensor, nil
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
