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

package controller

import (
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/blackrock/axis/common"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	executorEnvVars = []corev1.EnvVar{
		envFromField(common.EnvVarJobName, "metadata.name"),
		envFromField(common.EnvVarNamespace, "metadata.namespace"),
	}
)

// envFromField is a helper to return a EnvVar with the name and field
func envFromField(envVarName, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

func (soc *sOperationCtx) createSensorExecutorJob() error {
	envVars := append(executorEnvVars)

	//todo: completions could be a nice way to define explicity the repeatability of a sensor
	var completions int32
	completions = 1

	execContainer := corev1.Container{
		Name:            common.ExecutorContainerName,
		Image:           soc.controller.Config.ExecutorImage,
		Env:             envVars,
		ImagePullPolicy: corev1.PullIfNotPresent,
	}
	if soc.controller.Config.ExecutorResources != nil {
		execContainer.Resources = *soc.controller.Config.ExecutorResources
	}

	//todo: attach some PVC for storing the events
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.CreateJobPrefix(soc.s.Name),
			Labels: map[string]string{
				common.LabelKeyResolved: "false",
				common.LabelJobName: common.CreateJobPrefix(soc.s.Name),
			},
			Annotations: map[string]string{},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(soc.s, v1alpha1.SchemaGroupVersionKind),
			},
		},
		Spec: batchv1.JobSpec{
			Completions: &completions,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: common.CreateJobPrefix(soc.s.Name),
					Labels: map[string]string{
						common.LabelKeySensor:   soc.s.Name,
						common.LabelKeyResolved: "false",
					},
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(soc.s, v1alpha1.SchemaGroupVersionKind),
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: soc.controller.Config.ServiceAccount,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						execContainer,
					},
				},
			},
		},
	}
	if soc.controller.Config.InstanceID != "" {
		job.ObjectMeta.Labels[common.LabelKeySensorControllerInstanceID] = soc.controller.Config.InstanceID
		job.Spec.Template.ObjectMeta.Labels[common.LabelKeySensorControllerInstanceID] = soc.controller.Config.InstanceID
	}

	createdJob, err := soc.controller.kubeClientset.BatchV1().Jobs(soc.s.ObjectMeta.Namespace).Create(&job)
	if err != nil {
		soc.log.Warnf("Failed to create executor job", zap.Error(err))
		return err
	}
	soc.log.Infof("created signal executor job '%s'", createdJob.Name)

	// If signal is of type Webhook, create a service backed by the job
	for _, signal := range soc.s.Spec.Signals {
		if signal.GetType() == v1alpha1.SignalTypeWebhook {
			targetPort := signal.Webhook.Port
			if signal.Webhook.Port == 0 {
				targetPort = common.WebhookServiceTargetPort
			}
			webhookSvc := corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: common.CreateServiceSuffix(soc.s.Name),
					Namespace: soc.s.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(soc.s, v1alpha1.SchemaGroupVersionKind),
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Selector: map[string]string{
						common.LabelJobName: createdJob.ObjectMeta.Labels[common.LabelJobName],
					},
					Ports: []corev1.ServicePort{
						{
							Protocol: corev1.ProtocolTCP,
							Port: common.WebhookServicePort,
							TargetPort: intstr.FromInt(targetPort),
						},
					},
				},
			}
			createdSvc, err := soc.controller.kubeClientset.CoreV1().Services(soc.s.ObjectMeta.Namespace).Create(&webhookSvc)
			if err != nil {
				soc.log.Warnf("Failed to create executor service", zap.Error(err))
				return err
			}
			soc.log.Infof("Created executor service %s", createdSvc.Name)
		}
	}
	return nil
}