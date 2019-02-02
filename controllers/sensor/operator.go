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
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
}

// newSensorOperationCtx creates and initializes a new sOperationCtx object
func newSensorOperationCtx(s *v1alpha1.Sensor, controller *SensorController) *sOperationCtx {
	return &sOperationCtx{
		s:          s.DeepCopy(),
		updated:    false,
		log:        common.GetLoggerContext(common.LoggerConf()).Str("sensor-name", s.Name).Str("sensor-namespace", s.Namespace).Logger(),
		controller: controller,
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
		// perform one-time sensor validation
		// non nil err indicates failed validation
		// we do not want to requeue a sensor in this case
		// since validation will fail every time
		err := ValidateSensor(soc.s)
		if err != nil {
			soc.log.Error().Err(err).Msg("failed to validate sensor")
			soc.markSensorPhase(v1alpha1.NodePhaseError, true, err.Error())
			return nil
		}

		// Initialize all event dependency nodes
		for _, dependency := range soc.s.Spec.Dependencies {
			InitializeNode(soc.s, dependency.Name, v1alpha1.NodeTypeEventDependency, &soc.log)
		}

		// Initialize all trigger nodes
		for _, trigger := range soc.s.Spec.Triggers {
			InitializeNode(soc.s, trigger.Name, v1alpha1.NodeTypeTrigger, &soc.log)
		}

		// add default env variables
		soc.s.Spec.DeploySpec.Containers[0].Env = append(soc.s.Spec.DeploySpec.Containers[0].Env, []corev1.EnvVar{
			{
				Name:  common.SensorName,
				Value: soc.s.Name,
			},
			{
				Name:  common.SensorNamespace,
				Value: soc.s.Namespace,
			},
			{
				Name:  common.EnvVarSensorControllerInstanceID,
				Value: soc.controller.Config.InstanceID,
			},
		}...,
		)

		// create a sensor pod
		// default label on sensor
		if soc.s.ObjectMeta.Labels == nil {
			soc.s.ObjectMeta.Labels = make(map[string]string)
		}
		soc.s.ObjectMeta.Labels[common.LabelSensorName] = soc.s.Name

		sensorPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      soc.s.Name,
				Namespace: soc.s.Namespace,
				Labels:    soc.s.Labels,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(soc.s, v1alpha1.SchemaGroupVersionKind),
				},
			},
			Spec: *soc.s.Spec.DeploySpec,
		}
		_, err = soc.controller.kubeClientset.CoreV1().Pods(soc.s.Namespace).Create(sensorPod)
		if err != nil {
			soc.log.Error().Err(err).Msg("failed to create sensor pod")
			return err
		}
		soc.log.Info().Msg("sensor pod created")

		// Create a ClusterIP service to expose sensor in cluster if the event protocol type is HTTP
		if _, err = soc.controller.kubeClientset.CoreV1().Services(soc.s.Namespace).Get(common.DefaultServiceName(soc.s.Name), metav1.GetOptions{}); err != nil && apierr.IsNotFound(err) && soc.s.Spec.EventProtocol.Type == pc.HTTP {
			// Create sensor service
			sensorSvc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      common.DefaultServiceName(soc.s.Name),
					Namespace: soc.s.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(soc.s, v1alpha1.SchemaGroupVersionKind),
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port:       intstr.Parse(soc.s.Spec.EventProtocol.Http.Port).IntVal,
							TargetPort: intstr.FromInt(int(intstr.Parse(soc.s.Spec.EventProtocol.Http.Port).IntVal)),
						},
					},
					Type:     corev1.ServiceTypeClusterIP,
					Selector: soc.s.Labels,
				},
			}
			_, err = soc.controller.kubeClientset.CoreV1().Services(soc.s.Namespace).Create(sensorSvc)
			if err != nil {
				soc.log.Error().Err(err).Msg("failed to create sensor service")
				return err
			}
		}

		// Mark all eventDependency nodes as active
		for _, dependency := range soc.s.Spec.Dependencies {
			MarkNodePhase(soc.s, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, &soc.log, "node is active")
		}

		// if we get here - we know the signals are running
		soc.log.Info().Msg("marking sensor as active")
		soc.markSensorPhase(v1alpha1.NodePhaseActive, false, "listening for events")

	case v1alpha1.NodePhaseActive:
		soc.log.Info().Msg("sensor is already running")

	case v1alpha1.NodePhaseError:
		soc.log.Info().Msg("sensor is in error state. check sensor resource status information and corresponding escalated K8 event for the error")
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
