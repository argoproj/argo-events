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

	"github.com/imdario/mergo"

	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/pkg/errors"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// generateServiceSpec returns a K8s service spec for the sensor
func (ctx *sensorContext) generateServiceSpec() *corev1.Service {
	port := common.SensorServerPort
	if ctx.sensor.Spec.Subscription.HTTP != nil {
		port = ctx.sensor.Spec.Subscription.HTTP.Port
	}

	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-sensor-svc", ctx.sensor.Name),
			Labels:      ctx.sensor.Spec.ServiceLabels,
			Annotations: ctx.sensor.Spec.ServiceAnnotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       intstr.FromInt(port).IntVal,
					TargetPort: intstr.FromInt(port),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				common.LabelOwnerName: ctx.sensor.Name,
			},
		},
	}

	// Non-overrideable labels required by sensor service
	if serviceSpec.ObjectMeta.Labels == nil {
		serviceSpec.ObjectMeta.Labels = map[string]string{}
	}

	serviceSpec.ObjectMeta.Labels[common.LabelSensorName] = ctx.sensor.Name
	serviceSpec.ObjectMeta.Labels[LabelControllerInstanceID] = ctx.controller.Config.InstanceID

	return serviceSpec
}

// serviceBuilder builds a new service that exposes sensor.
func (ctx *sensorContext) serviceBuilder() (*corev1.Service, error) {
	service := ctx.generateServiceSpec()
	if err := controllerscommon.SetObjectMeta(ctx.sensor, service, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return service, nil
}

func (ctx *sensorContext) makeDeploymentSpec() appv1.DeploymentSpec {
	// Deprecated spec, will be unsupported soon.
	if ctx.sensor.Spec.Template.Spec != nil {
		ctx.logger.WithField("name", ctx.sensor.Name).WithField("namespace", ctx.sensor.Namespace).Warn("spec.template.spec is DEPRECATED, it will be unsupported soon, please use spec.template.container")
		return ctx.makeLegacyDeploymentSpec()
	}

	replicas := int32(1)
	labels := map[string]string{
		common.LabelObjectName: ctx.sensor.Name,
	}
	sensorContainer := corev1.Container{
		Image:           ctx.controller.sensorImage,
		ImagePullPolicy: corev1.PullAlways,
	}
	if ctx.sensor.Spec.Template.Container != nil {
		mergo.Merge(&sensorContainer, ctx.sensor.Spec.Template.Container, mergo.WithOverride)
	}
	sensorContainer.Name = "main"
	return appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: ctx.sensor.Spec.Template.ServiceAccountName,
				Containers: []corev1.Container{
					sensorContainer,
				},
				Volumes:         ctx.sensor.Spec.Template.Volumes,
				SecurityContext: ctx.sensor.Spec.Template.SecurityContext,
			},
		},
	}
}

// makeLegacyDeploymentSpec is deprecated, will be unsupported soon.
func (ctx *sensorContext) makeLegacyDeploymentSpec() appv1.DeploymentSpec {
	replicas := int32(1)
	labels := map[string]string{
		common.LabelObjectName: ctx.sensor.Name,
	}
	return appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: *ctx.sensor.Spec.Template.Spec,
		},
	}
}

// deploymentBuilder builds the deployment specification for the sensor
func (ctx *sensorContext) deploymentBuilder() (*appv1.Deployment, error) {
	deploymentSpec := ctx.makeDeploymentSpec()
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    ctx.sensor.Namespace,
			GenerateName: fmt.Sprintf("%s-sensor-", ctx.sensor.Name),
			Labels:       deploymentSpec.Template.Labels,
		},
		Spec: deploymentSpec,
	}
	envVars := []corev1.EnvVar{
		{
			Name:  common.SensorName,
			Value: ctx.sensor.Name,
		},
		{
			Name:  common.SensorNamespace,
			Value: ctx.sensor.Namespace,
		},
		{
			Name:  common.EnvVarControllerInstanceID,
			Value: ctx.controller.Config.InstanceID,
		},
	}
	for i, container := range deployment.Spec.Template.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		deployment.Spec.Template.Spec.Containers[i] = container
	}
	if err := controllerscommon.SetObjectMeta(ctx.sensor, deployment, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return deployment, nil
}

// createDeployment creates a deployment for the sensor
func (ctx *sensorContext) createDeployment(deployment *appv1.Deployment) (*appv1.Deployment, error) {
	return ctx.controller.k8sClient.AppsV1().Deployments(ctx.sensor.Namespace).Create(deployment)
}

// createService creates a service for the sensor
func (ctx *sensorContext) createService(service *corev1.Service) (*corev1.Service, error) {
	return ctx.controller.k8sClient.CoreV1().Services(ctx.sensor.Namespace).Create(service)
}

// updateDeployment updates the deployment for the sensor
func (ctx *sensorContext) updateDeployment() (*appv1.Deployment, error) {
	newDeployment, err := ctx.deploymentBuilder()
	if err != nil {
		return nil, err
	}

	currentMetadata := ctx.sensor.Status.Resources.Deployment
	if currentMetadata == nil {
		return nil, errors.New("deployment metadata is expected to be set in sensor object")
	}

	currentDeployment, err := ctx.controller.k8sClient.AppsV1().Deployments(ctx.sensor.Namespace).Get(currentMetadata.Name, metav1.GetOptions{})
	if err != nil {
		if apierror.IsNotFound(err) {
			ctx.updated = true
			return ctx.controller.k8sClient.AppsV1().Deployments(ctx.sensor.Namespace).Create(newDeployment)
		}
		return nil, err
	}

	if currentDeployment.Annotations != nil && currentDeployment.Annotations[common.AnnotationResourceSpecHash] != newDeployment.Annotations[common.AnnotationResourceSpecHash] {
		ctx.updated = true
		currentDeployment.Spec = newDeployment.Spec
		currentDeployment.Annotations[common.AnnotationResourceSpecHash] = newDeployment.Annotations[common.AnnotationResourceSpecHash]
		return ctx.controller.k8sClient.AppsV1().Deployments(ctx.sensor.Namespace).Update(currentDeployment)
	}
	return currentDeployment, nil
}

// updateService updates the service for the sensor
func (ctx *sensorContext) updateService() (*corev1.Service, error) {
	currentMetadata := ctx.sensor.Status.Resources.Service

	newService, err := ctx.serviceBuilder()
	if err != nil {
		return nil, err
	}

	if currentMetadata == nil {
		ctx.updated = true
		return ctx.controller.k8sClient.CoreV1().Services(newService.Namespace).Create(newService)
	}

	if currentMetadata.Annotations != nil && currentMetadata.Annotations[common.AnnotationResourceSpecHash] != newService.Annotations[common.AnnotationResourceSpecHash] {
		ctx.updated = true
		if err := ctx.controller.k8sClient.CoreV1().Services(currentMetadata.Namespace).Delete(currentMetadata.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
	}
	return ctx.controller.k8sClient.CoreV1().Services(currentMetadata.Namespace).Get(currentMetadata.Name, metav1.GetOptions{})
}
