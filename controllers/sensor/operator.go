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
	"github.com/sirupsen/logrus"
	"time"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// the context of an operation on a sensor.
// the sensor-controller creates this context each time it picks a Sensor off its queue.
type sOperationCtx struct {
	// s is the sensor object
	s *v1alpha1.Sensor
	// updated indicates whether the sensor object was updated and needs to be persisted back to k8
	updated bool
	// log is the logrus logging context to correlate logs with a sensor
	log *logrus.Logger
	// reference to the sensor-controller
	controller *SensorController
	// srctx is the context to handle child resource
	srctx sResourceCtx
}

// newSensorOperationCtx creates and initializes a new sOperationCtx object
func newSensorOperationCtx(s *v1alpha1.Sensor, controller *SensorController) *sOperationCtx {
	return &sOperationCtx{
		s:       s.DeepCopy(),
		updated: false,
		log: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelSensorName: s.Name,
				common.LabelNamespace:  s.Namespace,
			}).Logger,
		controller: controller,
		srctx:      NewSensorResourceContext(s, controller),
	}
}

// operate on sensor resource
func (soc *sOperationCtx) operate() error {
	defer func() {
		if soc.updated {
			// persist updates to sensor resource
			labels := map[string]string{
				common.LabelSensorName:                    soc.s.Name,
				common.LabelSensorKeyPhase:                string(soc.s.Status.Phase),
				common.LabelKeySensorControllerInstanceID: soc.controller.Config.InstanceID,
				common.LabelOperation:                     "persist_state_update",
			}
			eventType := common.StateChangeEventType

			updatedSensor, err := PersistUpdates(soc.controller.sensorClientset, soc.s, soc.controller.Config.InstanceID, soc.log)
			if err != nil {
				soc.log.WithError(err).Error("failed to persist sensor update, escalating...")

				// escalate failure
				eventType = common.EscalationEventType
			}

			// update sensor ref. in case of failure to persist updates, this is a deep copy of old sensor resource
			soc.s = updatedSensor

			labels[common.LabelEventType] = string(eventType)
			if err := common.GenerateK8sEvent(soc.controller.kubeClientset,
				"persist update",
				eventType,
				"sensor state update",
				soc.s.Name,
				soc.s.Namespace,
				soc.controller.Config.InstanceID,
				sensor.Kind,
				labels); err != nil {
				soc.log.WithError(err).Error("failed to create K8s event to log sensor state persist operation")
				return
			}
			soc.log.Info("successfully persisted sensor resource update and created K8s event")
		}
		soc.updated = false
	}()

	switch soc.s.Status.Phase {
	case v1alpha1.NodePhaseNew:
		err := soc.createSensorResources()
		if err != nil {
			return err
		}

	case v1alpha1.NodePhaseActive:
		soc.log.Info("sensor is running")

		err := soc.updateSensorResources()
		if err != nil {
			return err
		}

	case v1alpha1.NodePhaseError:
		soc.log.Info("sensor is in error state. check sensor resource status information and corresponding escalated K8 event for the error")

		err := soc.updateSensorResources()
		if err != nil {
			return err
		}
	}
	return nil
}

func (soc *sOperationCtx) createSensorResources() error {
	err := ValidateSensor(soc.s)
	if err != nil {
		soc.log.WithError(err).Error("failed to validate sensor")
		err = errors.Wrap(err, "failed to validate sensor")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		return err
	}

	soc.initializeAllNodes()
	pod, err := soc.createSensorPod()
	if err != nil {
		err = errors.Wrap(err, "failed to create sensor pod")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		return err
	}
	soc.markAllNodePhases()
	soc.log.WithField(common.LabelPodName, pod.Name).Info("sensor pod is created")

	// expose sensor if service is configured
	if soc.srctx.getServiceTemplateSpec() != nil {
		svc, err := soc.createSensorService()
		if err != nil {
			err = errors.Wrap(err, "failed to create sensor service")
			soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
			return err
		}
		soc.log.WithField(common.LabelServiceName, svc.Name).Info("sensor service is created")
	}

	// if we get here - we know the signals are running
	soc.log.Info("marking sensor as active")
	soc.markSensorPhase(v1alpha1.NodePhaseActive, false, "listening for events")
	return nil
}

func (soc *sOperationCtx) createSensorPod() (*corev1.Pod, error) {
	pod, err := soc.srctx.newSensorPod()
	if err != nil {
		soc.log.WithError(err).Error("failed to initialize pod for sensor")
		return nil, err
	}
	pod, err = soc.srctx.createSensorPod(pod)
	if err != nil {
		soc.log.WithError(err).Error("failed to create pod for sensor")
		return nil, err
	}
	return pod, nil
}

func (soc *sOperationCtx) createSensorService() (*corev1.Service, error) {
	svc, err := soc.srctx.newSensorService()
	if err != nil {
		soc.log.WithError(err).Error("failed to initialize service for sensor")
		return nil, err
	}
	svc, err = soc.srctx.createSensorService(svc)
	if err != nil {
		soc.log.WithError(err).Error("failed to create service for sensor")
		return nil, err
	}
	return svc, nil
}

func (soc *sOperationCtx) updateSensorResources() error {
	err := ValidateSensor(soc.s)
	if err != nil {
		soc.log.WithError(err).Error("failed to validate sensor")
		err = errors.Wrap(err, "failed to validate sensor")
		if soc.s.Status.Phase != v1alpha1.NodePhaseError {
			soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		}
		return err
	}

	_, podChanged, err := soc.updateSensorPod()
	if err != nil {
		err = errors.Wrap(err, "failed to update sensor pod")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		return err
	}

	_, svcChanged, err := soc.updateSensorService()
	if err != nil {
		err = errors.Wrap(err, "failed to update sensor service")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		return err
	}

	if soc.s.Status.Phase != v1alpha1.NodePhaseActive && (podChanged || svcChanged) {
		soc.markSensorPhase(v1alpha1.NodePhaseActive, false, "sensor is active")
	}

	return nil
}

func (soc *sOperationCtx) updateSensorPod() (*corev1.Pod, bool, error) {
	// Check if sensor spec has changed for pod.
	existingPod, err := soc.srctx.getSensorPod()
	if err != nil {
		soc.log.WithError(err).Error("failed to get pod for sensor")
		return nil, false, err
	}

	// create a new pod spec
	newPod, err := soc.srctx.newSensorPod()
	if err != nil {
		soc.log.WithError(err).Error("failed to initialize pod for sensor")
		return nil, false, err
	}

	// check if pod spec remained unchanged
	if existingPod != nil {
		if existingPod.Annotations != nil && existingPod.Annotations[common.AnnotationSensorResourceSpecHashName] == newPod.Annotations[common.AnnotationSensorResourceSpecHashName] {
			soc.log.WithField(common.LabelPodName, existingPod.Name).Debug("sensor pod spec unchanged")
			return nil, false, nil
		}

		// By now we are sure that the spec changed, so lets go ahead and delete the exisitng sensor pod.
		soc.log.WithField(common.LabelPodName, existingPod.Name).Info("sensor pod spec changed")

		err := soc.srctx.deleteSensorPod(existingPod)
		if err != nil {
			soc.log.WithError(err).Error("failed to delete pod for sensor")
			return nil, false, err
		}

		soc.log.WithField(common.LabelPodName, existingPod.Name).Info("sensor pod is deleted")
	}

	// Create new pod for updated sensor spec.
	createdPod, err := soc.srctx.createSensorPod(newPod)
	if err != nil {
		soc.log.WithError(err).Error("failed to create pod for sensor")
		return nil, false, err
	}
	soc.log.WithField(common.LabelPodName, newPod.Name).Info("sensor pod is created")

	return createdPod, true, nil
}

func (soc *sOperationCtx) updateSensorService() (*corev1.Service, bool, error) {
	// Check if sensor spec has changed for service.
	existingSvc, err := soc.srctx.getSensorService()
	if err != nil {
		soc.log.WithError(err).Error("failed to get service for sensor")
		return nil, false, err
	}

	// create a new service spec
	newSvc, err := soc.srctx.newSensorService()
	if err != nil {
		soc.log.WithError(err).Error("failed to initialize service for sensor")
		return nil, false, err
	}

	if existingSvc != nil {
		// updated spec doesn't have service defined, delete existing service.
		if newSvc == nil {
			if err := soc.srctx.deleteSensorService(existingSvc); err != nil {
				return nil, false, err
			}
			return nil, true, nil
		}

		// check if service spec remained unchanged
		if existingSvc.Annotations[common.AnnotationSensorResourceSpecHashName] == newSvc.Annotations[common.AnnotationSensorResourceSpecHashName] {
			soc.log.WithField(common.LabelServiceName, existingSvc.Name).Debug("sensor service spec unchanged")
			return nil, false, nil
		}

		// service spec changed, delete existing service and create new one
		soc.log.WithField(common.LabelServiceName, existingSvc.Name).Info("sensor service spec changed")

		if err := soc.srctx.deleteSensorService(existingSvc); err != nil {
			return nil, false, err
		}
	} else if newSvc == nil {
		// sensor service doesn't exist originally
		return nil, false, nil
	}

	// change createSensorService to take a service spec
	createdSvc, err := soc.srctx.createSensorService(newSvc)
	if err != nil {
		soc.log.WithField(common.LabelServiceName, newSvc.Name).WithError(err).Error("failed to create service for sensor")
		return nil, false, err
	}
	soc.log.WithField(common.LabelServiceName, newSvc.Name).Info("sensor service is created")

	return createdSvc, true, nil
}

// mark the overall sensor phase
func (soc *sOperationCtx) markSensorPhase(phase v1alpha1.NodePhase, markComplete bool, message ...string) {
	justCompleted := soc.s.Status.Phase != phase
	if justCompleted {
		soc.log.WithFields(
			map[string]interface{}{
				"old": string(soc.s.Status.Phase),
				"new": string(phase),
			},
		).Info("phase updated")

		soc.s.Status.Phase = phase
		if soc.s.ObjectMeta.Labels == nil {
			soc.s.ObjectMeta.Labels = make(map[string]string)
		}
		if soc.s.ObjectMeta.Annotations == nil {
			soc.s.ObjectMeta.Annotations = make(map[string]string)
		}
		soc.s.ObjectMeta.Labels[common.LabelSensorKeyPhase] = string(phase)
		// add annotations so a resource sensor can watch this sensor.
		soc.s.ObjectMeta.Annotations[common.LabelSensorKeyPhase] = string(phase)
	}
	if soc.s.Status.StartedAt.IsZero() {
		soc.s.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}
	if len(message) > 0 && soc.s.Status.Message != message[0] {
		soc.log.WithFields(
			map[string]interface{}{
				"old": soc.s.Status.Message,
				"new": message[0],
			},
		).Info("sensor message updated")
		soc.s.Status.Message = message[0]
	}

	switch phase {
	case v1alpha1.NodePhaseComplete, v1alpha1.NodePhaseError:
		if markComplete && justCompleted {
			soc.log.Info("marking sensor complete")
			soc.s.Status.CompletedAt = metav1.Time{Time: time.Now().UTC()}
			if soc.s.ObjectMeta.Labels == nil {
				soc.s.ObjectMeta.Labels = make(map[string]string)
			}
			soc.s.ObjectMeta.Labels[common.LabelSensorKeyComplete] = "true"
			soc.s.ObjectMeta.Annotations[common.LabelSensorKeyComplete] = string(phase)
		}
	}
	soc.updated = true
}

func (soc *sOperationCtx) initializeAllNodes() {
	// Initialize all event dependency nodes
	for _, dependency := range soc.s.Spec.Dependencies {
		InitializeNode(soc.s, dependency.Name, v1alpha1.NodeTypeEventDependency, soc.log)
	}

	// Initialize all dependency groups
	if soc.s.Spec.DependencyGroups != nil {
		for _, group := range soc.s.Spec.DependencyGroups {
			InitializeNode(soc.s, group.Name, v1alpha1.NodeTypeDependencyGroup, soc.log)
		}
	}

	// Initialize all trigger nodes
	for _, trigger := range soc.s.Spec.Triggers {
		InitializeNode(soc.s, trigger.Template.Name, v1alpha1.NodeTypeTrigger, soc.log)
	}
}

func (soc *sOperationCtx) markAllNodePhases() {
	// Mark all event dependency nodes as active
	for _, dependency := range soc.s.Spec.Dependencies {
		MarkNodePhase(soc.s, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, soc.log, "node is active")
	}

	// Mark all dependency groups as active
	if soc.s.Spec.DependencyGroups != nil {
		for _, group := range soc.s.Spec.DependencyGroups {
			MarkNodePhase(soc.s, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, soc.log, "node is active")
		}
	}
}
