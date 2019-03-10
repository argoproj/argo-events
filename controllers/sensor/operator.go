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
	"fmt"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/rs/zerolog"
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
	log zerolog.Logger
	// reference to the sensor-controller
	controller *SensorController
	// srctx is the context to handle child resource
	srctx sResourceCtx
}

// newSensorOperationCtx creates and initializes a new sOperationCtx object
func newSensorOperationCtx(s *v1alpha1.Sensor, controller *SensorController) *sOperationCtx {
	return &sOperationCtx{
		s:          s.DeepCopy(),
		updated:    false,
		log:        common.GetLoggerContext(common.LoggerConf()).Str("sensor-name", s.Name).Str("sensor-namespace", s.Namespace).Logger(),
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

			updatedSensor, err := PersistUpdates(soc.controller.sensorClientset, soc.s, soc.controller.Config.InstanceID, &soc.log)
			if err != nil {
				soc.log.Error().Err(err).Msg("failed to persist sensor update, escalating...")

				// escalate failure
				eventType = common.EscalationEventType
			}

			// update sensor ref. in case of failure to persist updates, this is a deep copy of old sensor resource
			soc.s = updatedSensor

			labels[common.LabelEventType] = string(eventType)
			if err := common.GenerateK8sEvent(soc.controller.kubeClientset, "persist update", eventType, "sensor state update", soc.s.Name,
				soc.s.Namespace, soc.controller.Config.InstanceID, sensor.Kind, labels); err != nil {
				soc.log.Error().Err(err).Msg("failed to create K8s event to log sensor state persist operation")
				return
			}
			soc.log.Info().Msg("successfully persisted sensor resource update and created K8s event")
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
		soc.log.Info().Msg("sensor is running")

		err := soc.updateSensorResources()
		if err != nil {
			return err
		}

	case v1alpha1.NodePhaseError:
		soc.log.Info().Msg("sensor is in error state. check sensor resource status information and corresponding escalated K8 event for the error")

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
		soc.log.Error().Err(err).Msg("failed to validate sensor")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		return nil
	}

	soc.initializeAllNodes()
	pod, err := soc.srctx.createSensorPod()
	if err != nil {
		soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to create pod for sensor")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to create sensor pod. err: %s", err))
		return err
	}
	soc.markAllNodePhases()
	soc.log.Info().Str("pod-name", pod.Name).Msg("sensor pod is created")

	// expose sensor if service is configured
	if soc.srctx.getServiceSpec() != nil {
		svc, err := soc.srctx.createSensorService()
		if err != nil {
			soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to create service for sensor")
			soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to create sensor service. err: %s", err))
			return err
		}
		soc.log.Info().Str("svc-name", svc.Name).Msg("sensor service is created")
	}

	// if we get here - we know the signals are running
	soc.log.Info().Msg("marking sensor as active")
	soc.markSensorPhase(v1alpha1.NodePhaseActive, false, "listening for events")
	return nil
}

func (soc *sOperationCtx) updateSensorResources() error {
	err := ValidateSensor(soc.s)
	if err != nil {
		soc.log.Error().Err(err).Msg("failed to validate sensor")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
		return nil
	}

	created := false
	deleted := false

	pod, err := soc.srctx.getSensorPod()
	if err != nil {
		soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to get pod for sensor")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to get sensor pod. err: %s", err))
		return err
	}
	if pod != nil {
		newPod, err := soc.srctx.newSensorPod()
		if err != nil {
			soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to initialize pod for sensor")
			soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to initialize sensor pod. err: %s", err))
			return err
		}
		if pod.Annotations == nil || pod.Annotations[common.AnnotationSensorResourceSpecHashName] != newPod.Annotations[common.AnnotationSensorResourceSpecHashName] {
			soc.log.Info().Str("pod-name", pod.Name).Msg("sensor pod spec changed")
			err := soc.controller.kubeClientset.CoreV1().Pods(soc.s.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			if err != nil {
				soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to delete pod for sensor")
				soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to delete sensor pod. err: %s", err))
				return err
			}
			soc.log.Info().Str("pod-name", pod.Name).Msg("sensor pod is deleted")
			deleted = true
		}
	} else {
		soc.log.Info().Str("sensor-name", soc.s.Name).Msg("sensor pod has been deleted")
		soc.initializeAllNodes()
		pod, err := soc.srctx.createSensorPod()
		if err != nil {
			soc.log.Error().Err(err).Msg("failed to create pod for sensor")
			soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to create sensor pod. err: %s", err))
			return err
		}
		soc.markAllNodePhases()
		soc.log.Info().Str("pod-name", pod.Name).Msg("sensor pod is created")
		created = true
	}

	svc, err := soc.srctx.getSensorService()
	if err != nil {
		soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to get service for sensor")
		soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to get sensor service. err: %s", err))
		return err
	}
	if svc != nil {
		var newSvc *corev1.Service
		if soc.srctx.getServiceSpec() != nil {
			newSvc, err = soc.srctx.newSensorService()
			if err != nil {
				soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to initialize service for sensor")
				soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to initialize sensor service. err: %s", err))
				return err
			}
		}
		if newSvc == nil || svc.Annotations == nil || svc.Annotations[common.AnnotationSensorResourceSpecHashName] != newSvc.Annotations[common.AnnotationSensorResourceSpecHashName] {
			soc.log.Info().Str("svc-name", svc.Name).Msg("sensor service spec changed")
			err := soc.controller.kubeClientset.CoreV1().Services(soc.s.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
			if err != nil {
				soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to delete service for sensor")
				soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to delete sensor service. err: %s", err))
				return err
			}
			soc.log.Info().Str("svc-name", svc.Name).Msg("sensor service is deleted")
			deleted = true
		}
	} else {
		if soc.srctx.getServiceSpec() != nil {
			soc.log.Info().Str("sensor-name", soc.s.Name).Msg("sensor service has been deleted")
			svc, err := soc.srctx.createSensorService()
			if err != nil {
				soc.log.Error().Err(err).Str("sensor-name", soc.s.Name).Msg("failed to create service for sensor")
				soc.markSensorPhase(v1alpha1.NodePhaseError, false, fmt.Sprintf("failed to create sensor service. err: %s", err))
				return err
			}
			soc.log.Info().Str("svc-name", svc.Name).Msg("sensor service is created")
			created = true
		}
	}

	if created && !deleted && soc.s.Status.Phase != v1alpha1.NodePhaseActive {
		soc.markSensorPhase(v1alpha1.NodePhaseActive, false, "sensor is active")
	}
	return nil
}

// mark the overall sensor phase
func (soc *sOperationCtx) markSensorPhase(phase v1alpha1.NodePhase, markComplete bool, message ...string) {
	justCompleted := soc.s.Status.Phase != phase
	if justCompleted {
		soc.log.Info().Str("old-phase", string(soc.s.Status.Phase)).Str("new-phase", string(phase)).Msg("sensor phase updated")
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
		soc.log.Info().Str("old-message", soc.s.Status.Message).Str("new-message", message[0]).Msg("sensor message updated")
		soc.s.Status.Message = message[0]
	}

	switch phase {
	case v1alpha1.NodePhaseComplete, v1alpha1.NodePhaseError:
		if markComplete && justCompleted {
			soc.log.Info().Msg("marking sensor complete")
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
		InitializeNode(soc.s, dependency.Name, v1alpha1.NodeTypeEventDependency, &soc.log)
	}

	// Initialize all dependency groups
	if soc.s.Spec.DependencyGroups != nil {
		for _, group := range soc.s.Spec.DependencyGroups {
			InitializeNode(soc.s, group.Name, v1alpha1.NodeTypeDependencyGroup, &soc.log)
		}
	}

	// Initialize all trigger nodes
	for _, trigger := range soc.s.Spec.Triggers {
		InitializeNode(soc.s, trigger.Name, v1alpha1.NodeTypeTrigger, &soc.log)
	}
}

func (soc *sOperationCtx) markAllNodePhases() {
	// Mark all event dependency nodes as active
	for _, dependency := range soc.s.Spec.Dependencies {
		MarkNodePhase(soc.s, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, &soc.log, "node is active")
	}

	// Mark all dependency groups as active
	if soc.s.Spec.DependencyGroups != nil {
		for _, group := range soc.s.Spec.DependencyGroups {
			MarkNodePhase(soc.s, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, &soc.log, "node is active")
		}
	}
}
