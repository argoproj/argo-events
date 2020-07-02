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

package gateway

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/imdario/mergo"

	"github.com/pkg/errors"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
)

// buildServiceResource builds a new service that exposes gateway.
func (ctx *gatewayContext) buildServiceResource() (*corev1.Service, error) {
	if ctx.gateway.Spec.Service == nil {
		return nil, nil
	}
	if ctx.gateway.Spec.Service.Spec != nil {
		// Deprecated spec, will be unsupported soon.
		ctx.logger.WithField("name", ctx.gateway.Name).WithField("namespace", ctx.gateway.Namespace).Warn("spec.service.spec is DEPRECATED, it will be unsupported soon, please use spec.service.ports")
		return ctx.buildLegacyServiceResource()
	}
	if len(ctx.gateway.Spec.Service.Ports) == 0 {
		return nil, nil
	}
	labels := map[string]string{
		common.LabelObjectName:  ctx.gateway.Name,
		common.LabelGatewayName: ctx.gateway.Name,
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-gateway", ctx.gateway.Name),
		},
		Spec: corev1.ServiceSpec{
			Ports:     ctx.gateway.Spec.Service.Ports,
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: ctx.gateway.Spec.Service.ClusterIP,
			Selector:  labels,
		},
	}
	if err := controllerscommon.SetObjectMeta(ctx.gateway, svc, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return svc, nil
}

// buildLegacyServiceResource is deprecated, will be unsupported soon.
func (ctx *gatewayContext) buildLegacyServiceResource() (*corev1.Service, error) {
	labels := map[string]string{
		common.LabelObjectName:  ctx.gateway.Name,
		common.LabelGatewayName: ctx.gateway.Name,
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-gateway", ctx.gateway.Name),
		},
		Spec: *ctx.gateway.Spec.Service.Spec,
	}
	svc.Spec.Selector = labels
	if err := controllerscommon.SetObjectMeta(ctx.gateway, svc, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return svc, nil
}

func (ctx *gatewayContext) makeDeploymentSpec() (*appv1.DeploymentSpec, error) {
	// Deprecated spec, will be unsupported soon.
	if ctx.gateway.Spec.Template.Spec != nil {
		ctx.logger.WithField("name", ctx.gateway.Name).WithField("namespace", ctx.gateway.Namespace).Warn("spec.template.spec is DEPRECATED, it will be unsupported soon, please use spec.template.container")
		return ctx.makeLegacyDeploymentSpec()
	}

	replicas := ctx.gateway.Spec.Replica
	if replicas == 0 {
		replicas = 1
	}

	labels := map[string]string{
		common.LabelGatewayName: ctx.gateway.Name,
		common.LabelObjectName:  ctx.gateway.Name,
	}
	podTemplateLabels := make(map[string]string)
	if len(ctx.gateway.Spec.Template.Metadata.Labels) > 0 {
		podTemplateLabels = ctx.gateway.Spec.Template.Metadata.Labels
	}
	for k, v := range labels {
		podTemplateLabels[k] = v
	}

	eventContainer := corev1.Container{
		Name:            "main",
		Image:           ctx.controller.serverImage,
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"/bin/gateway-server"},
		Args:            []string{string(ctx.gateway.Spec.Type)},
	}

	if ctx.gateway.Spec.Template.Container != nil {
		if err := mergo.Merge(&eventContainer, ctx.gateway.Spec.Template.Container, mergo.WithOverride); err != nil {
			return nil, err
		}
	}

	return &appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      podTemplateLabels,
				Annotations: ctx.gateway.Spec.Template.Metadata.Annotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: ctx.gateway.Spec.Template.ServiceAccountName,
				Containers: []corev1.Container{
					{
						Name:            "gateway-client",
						Image:           ctx.controller.clientImage,
						ImagePullPolicy: corev1.PullAlways,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    apiresource.MustParse("5m"),
								corev1.ResourceMemory: apiresource.MustParse("10Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    apiresource.MustParse("50m"),
								corev1.ResourceMemory: apiresource.MustParse("128Mi"),
							},
						},
					},
					eventContainer,
				},
				Affinity:        ctx.gateway.Spec.Template.Affinity,
				Tolerations:     ctx.gateway.Spec.Template.Tolerations,
				Volumes:         ctx.gateway.Spec.Template.Volumes,
				SecurityContext: ctx.gateway.Spec.Template.SecurityContext,
			},
		},
	}, nil
}

// makeLegacyDeploymentSpec is deprecated, will be unsupported soon.
func (ctx *gatewayContext) makeLegacyDeploymentSpec() (*appv1.DeploymentSpec, error) {
	replicas := ctx.gateway.Spec.Replica
	if replicas == 0 {
		replicas = 1
	}
	labels := map[string]string{
		common.LabelGatewayName: ctx.gateway.Name,
		common.LabelObjectName:  ctx.gateway.Name,
	}
	podTemplateLabels := make(map[string]string)
	if len(ctx.gateway.Spec.Template.Metadata.Labels) > 0 {
		podTemplateLabels = ctx.gateway.Spec.Template.Metadata.Labels
	}
	for k, v := range labels {
		podTemplateLabels[k] = v
	}

	return &appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      podTemplateLabels,
				Annotations: ctx.gateway.Spec.Template.Metadata.Annotations,
			},
			Spec: *ctx.gateway.Spec.Template.Spec,
		},
	}, nil
}

// buildDeploymentResource builds a deployment resource for the gateway
func (ctx *gatewayContext) buildDeploymentResource() (*appv1.Deployment, error) {
	// TODO: temporarily hardcode eventbus name here
	eventBusName := "default"
	if len(ctx.gateway.Spec.EventBusName) > 0 {
		eventBusName = ctx.gateway.Spec.EventBusName
	}
	eventBus, err := ctx.controller.eventBusClient.ArgoprojV1alpha1().EventBus(ctx.gateway.Namespace).Get(eventBusName, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			return nil, errors.Errorf("eventbus %s is not found in namespace %s", eventBusName, ctx.gateway.Namespace)
		}
		return nil, err
	}
	if !eventBus.Status.IsReady() {
		return nil, errors.Errorf("eventbus %s not ready in namespace %s", eventBusName, ctx.gateway.Namespace)
	}

	if ctx.gateway.Status.Resources == nil {
		ctx.gateway.Status.Resources = &v1alpha1.GatewayResource{}
	}

	deploymentSpec, err := ctx.makeDeploymentSpec()
	if err != nil {
		return nil, err
	}
	if eventBus.Status.Config.NATS != nil {
		natsConf := eventBus.Status.Config.NATS
		if natsConf.Auth != nil && natsConf.AccessSecret != nil {
			// Mount the secret as volume instead of using evnFrom to gain the ability
			// for the sensor deployment to auto reload when the secret changes
			volumes := deploymentSpec.Template.Spec.Volumes
			volumes = append(volumes, corev1.Volume{
				Name: "auth-volume",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: natsConf.AccessSecret.Name,
						Items: []corev1.KeyToPath{
							{
								Key:  natsConf.AccessSecret.Key,
								Path: "auth.yaml",
							},
						},
					},
				},
			})
			deploymentSpec.Template.Spec.Volumes = volumes
			for i, container := range deploymentSpec.Template.Spec.Containers {
				volumeMounts := container.VolumeMounts
				volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "auth-volume", MountPath: common.EventBusAuthFileMountPath})
				container.VolumeMounts = volumeMounts
				deploymentSpec.Template.Spec.Containers[i] = container
			}
		}
	} else {
		return nil, errors.New("unsupported event bus")
	}

	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    ctx.gateway.Namespace,
			GenerateName: fmt.Sprintf("%s-gateway-", ctx.gateway.Name),
			Labels:       deploymentSpec.Template.Labels,
		},
		Spec: *deploymentSpec,
	}

	processorPort := ctx.gateway.Spec.ProcessorPort
	if processorPort == "" {
		processorPort = common.GatewayProcessorPort
	}

	envVars := []corev1.EnvVar{
		{
			Name:  common.EnvVarNamespace,
			Value: ctx.gateway.Namespace,
		},
		{
			Name:  common.EnvVarEventSource,
			Value: ctx.gateway.Spec.EventSourceRef.Name,
		},
		{
			Name:  common.EnvVarResourceName,
			Value: ctx.gateway.Name,
		},
		{
			Name:  common.EnvVarControllerInstanceID,
			Value: ctx.controller.Config.InstanceID,
		},
		{
			Name:  common.EnvVarGatewayServerPort,
			Value: processorPort,
		},
		{
			Name:      "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:  common.EnvVarEventBusSubject,
			Value: fmt.Sprintf("eventbus-%s", ctx.gateway.Namespace),
		},
	}
	busConfigBytes, err := json.Marshal(eventBus.Status.Config)
	if err != nil {
		return nil, errors.Errorf("failed marshal event bus config: %v", err)
	}
	encodedBusConfig := base64.StdEncoding.EncodeToString(busConfigBytes)
	envVars = append(envVars, corev1.EnvVar{Name: common.EnvVarEventBusConfig, Value: encodedBusConfig})

	for i, container := range deployment.Spec.Template.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		deployment.Spec.Template.Spec.Containers[i] = container
	}

	if err := controllerscommon.SetObjectMeta(ctx.gateway, deployment, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, errors.Wrap(err, "failed to set the object metadata on the deployment object")
	}

	return deployment, nil
}

// createGatewayResources creates gateway deployment and service
func (ctx *gatewayContext) createGatewayResources() error {
	deployment, err := ctx.createGatewayDeployment()
	if err != nil {
		return err
	}
	ctx.gateway.Status.Resources.Deployment = &deployment.ObjectMeta
	ctx.logger.WithField("name", deployment.Name).WithField("namespace", deployment.Namespace).Infoln("gateway deployment is created")

	service, err := ctx.createGatewayService()
	if err != nil {
		return err
	}
	if service == nil {
		return nil
	}
	ctx.gateway.Status.Resources.Service = &service.ObjectMeta
	ctx.logger.WithField("name", service.Name).WithField("namespace", service.Namespace).Infoln("gateway service is created")

	return nil
}

// createGatewayDeployment creates a deployment for the gateway
func (ctx *gatewayContext) createGatewayDeployment() (*appv1.Deployment, error) {
	deployment, err := ctx.buildDeploymentResource()
	if err != nil {
		return nil, err
	}
	return ctx.controller.k8sClient.AppsV1().Deployments(ctx.gateway.Namespace).Create(deployment)
}

// createGatewayService creates a service for the gateway
func (ctx *gatewayContext) createGatewayService() (*corev1.Service, error) {
	svc, err := ctx.buildServiceResource()
	if err != nil {
		return nil, err
	}
	if svc == nil {
		return nil, nil
	}
	return ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).Create(svc)
}

// updateGatewayResources updates gateway deployment and service
func (ctx *gatewayContext) updateGatewayResources() error {
	deployment, err := ctx.updateGatewayDeployment()
	if err != nil {
		return err
	}
	if deployment != nil {
		ctx.gateway.Status.Resources.Deployment = &deployment.ObjectMeta
		ctx.logger.WithField("name", deployment.Name).WithField("namespace", deployment.Namespace).Infoln("gateway deployment is updated")
	}

	service, err := ctx.updateGatewayService()
	if err != nil {
		return err
	}
	if service != nil {
		ctx.gateway.Status.Resources.Service = &service.ObjectMeta
		ctx.logger.WithField("name", service.Name).WithField("namespace", service.Namespace).Infoln("gateway service is updated")
		return nil
	}
	ctx.gateway.Status.Resources.Service = nil
	return nil
}

// updateGatewayDeployment updates the gateway deployment
func (ctx *gatewayContext) updateGatewayDeployment() (*appv1.Deployment, error) {
	newDeployment, err := ctx.buildDeploymentResource()
	if err != nil {
		return nil, err
	}

	currentMetadata := ctx.gateway.Status.Resources.Deployment
	if currentMetadata == nil {
		return nil, errors.New("deployment metadata is expected to be set in gateway object")
	}

	currentDeployment, err := ctx.controller.k8sClient.AppsV1().Deployments(ctx.gateway.Namespace).Get(currentMetadata.Name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			ctx.updated = true
			return ctx.controller.k8sClient.AppsV1().Deployments(ctx.gateway.Namespace).Create(newDeployment)
		}
		return nil, err
	}

	if currentDeployment.Annotations != nil && currentDeployment.Annotations[common.AnnotationResourceSpecHash] != newDeployment.Annotations[common.AnnotationResourceSpecHash] {
		ctx.updated = true
		currentDeployment.Spec = newDeployment.Spec
		currentDeployment.Annotations[common.AnnotationResourceSpecHash] = newDeployment.Annotations[common.AnnotationResourceSpecHash]
		return ctx.controller.k8sClient.AppsV1().Deployments(ctx.gateway.Namespace).Update(currentDeployment)
	}

	return nil, nil
}

// updateGatewayService updates the gateway service
func (ctx *gatewayContext) updateGatewayService() (*corev1.Service, error) {
	newService, err := ctx.buildServiceResource()
	if err != nil {
		return nil, err
	}
	if newService == nil && ctx.gateway.Status.Resources.Service != nil {
		ctx.updated = true
		if err := ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).Delete(ctx.gateway.Status.Resources.Service.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if newService == nil {
		return nil, nil
	}

	if ctx.gateway.Status.Resources.Service == nil {
		ctx.updated = true
		return ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).Create(newService)
	}

	currentMetadata := ctx.gateway.Status.Resources.Service
	currentService, err := ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).Get(currentMetadata.Name, metav1.GetOptions{})
	if err != nil {
		ctx.updated = true
		return ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).Create(newService)
	}

	if currentMetadata == nil {
		return nil, errors.New("service metadata is expected to be set in gateway object")
	}

	if currentService.Annotations != nil && currentService.Annotations[common.AnnotationResourceSpecHash] != newService.Annotations[common.AnnotationResourceSpecHash] {
		ctx.updated = true
		if err := ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).Delete(currentMetadata.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		return ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Namespace).Create(newService)
	}

	return currentService, nil
}
