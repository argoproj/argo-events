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
	"fmt"

	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/pkg/errors"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildServiceResource builds a new service that exposes gateway.
func (ctx *gatewayContext) buildServiceResource() (*corev1.Service, error) {
	var svc *corev1.Service
	if template, ok := ctx.controller.TemplatesConfig[ctx.gateway.Spec.Type]; ok {
		if template.Service != nil {
			svc = template.Service.DeepCopy()
		}
	}
	if ctx.gateway.Spec.Service != nil {
		svc = common.CopyServiceSpec(svc, ctx.gateway.Spec.Service)
	}
	if svc == nil {
		return nil, nil
	}
	if len(svc.Name) == 0 {
		svc.Name = fmt.Sprintf("%s-gateway-svc", ctx.gateway.ObjectMeta.Name)
	}
	if err := controllerscommon.SetObjectMeta(ctx.gateway, svc, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	if svc.Spec.Selector == nil {
		svc.Spec.Selector = make(map[string]string)
	}
	svc.Spec.Selector[common.LabelGatewayName] = ctx.gateway.Name
	return svc, nil
}

// buildDeploymentResource builds a deployment resource for the gateway
func (ctx *gatewayContext) buildDeploymentResource() (*appv1.Deployment, error) {
	var podTemplate *corev1.PodTemplateSpec
	if template, ok := ctx.controller.TemplatesConfig[ctx.gateway.Spec.Type]; ok {
		if template.Deployment != nil {
			podTemplate = template.Deployment.DeepCopy()
		}
	}
	if ctx.gateway.Spec.Template != nil {
		podTemplate = common.CopyDeploymentSpecTemplate(podTemplate, ctx.gateway.Spec.Template)
	}
	if podTemplate == nil {
		return nil, errors.New("gateway template can't be empty")
	}
	// use an unique name to avoid potential conflict with existing deployments, also indicates it is an auto-generated deployment
	podTemplate.ObjectMeta.GenerateName = fmt.Sprintf("%s-gateway-%s-", ctx.gateway.Spec.Type, ctx.gateway.ObjectMeta.Name)
	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}
	podTemplate.Labels[common.LabelGatewayName] = ctx.gateway.Name

	replica := int32(ctx.gateway.Spec.Replica)
	if replica == 0 {
		replica = 1
	}

	deployment := &appv1.Deployment{
		ObjectMeta: podTemplate.ObjectMeta,
		Spec: appv1.DeploymentSpec{
			Replicas: &replica,
			Template: *podTemplate,
		},
	}

	if deployment.Spec.Template.Labels == nil {
		deployment.Spec.Template.Labels = map[string]string{}
	}
	deployment.Spec.Template.Labels[common.LabelObjectName] = ctx.gateway.Name

	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{},
		}
	}
	deployment.Spec.Selector.MatchLabels[common.LabelObjectName] = ctx.gateway.Name

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
	}

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
	if ctx.gateway.Status.Resources == nil {
		ctx.gateway.Status.Resources = &v1alpha1.GatewayResource{}
	}

	deployment, err := ctx.createGatewayDeployment()
	if err != nil {
		return err
	}
	ctx.gateway.Status.Resources.Deployment = &deployment.ObjectMeta
	ctx.logger.WithField("name", deployment.Name).WithField("namespace", deployment.Namespace).Infoln("gateway deployment is created")

	if ctx.gateway.Spec.Service != nil {
		service, err := ctx.createGatewayService()
		if err != nil {
			return err
		}
		ctx.gateway.Status.Resources.Service = &service.ObjectMeta
		ctx.logger.WithField("name", service.Name).WithField("namespace", service.Namespace).Infoln("gateway service is created")
	}

	return nil
}

// createGatewayDeployment creates a deployment for the gateway
func (ctx *gatewayContext) createGatewayDeployment() (*appv1.Deployment, error) {
	deployment, err := ctx.buildDeploymentResource()
	if err != nil {
		return nil, err
	}
	return ctx.controller.k8sClient.AppsV1().Deployments(deployment.Namespace).Create(deployment)
}

// createGatewayService creates a service for the gateway
func (ctx *gatewayContext) createGatewayService() (*corev1.Service, error) {
	svc, err := ctx.buildServiceResource()
	if err != nil {
		return nil, err
	}
	return ctx.controller.k8sClient.CoreV1().Services(svc.Namespace).Create(svc)
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

	currentDeployment, err := ctx.controller.k8sClient.AppsV1().Deployments(currentMetadata.Namespace).Get(currentMetadata.Name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			ctx.updated = true
			return ctx.controller.k8sClient.AppsV1().Deployments(newDeployment.Namespace).Create(newDeployment)
		}
		return nil, err
	}

	if currentDeployment.Annotations != nil && currentDeployment.Annotations[common.AnnotationResourceSpecHash] != newDeployment.Annotations[common.AnnotationResourceSpecHash] {
		ctx.updated = true
		if err := ctx.controller.k8sClient.AppsV1().Deployments(currentDeployment.Namespace).Delete(currentDeployment.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		return ctx.controller.k8sClient.AppsV1().Deployments(newDeployment.Namespace).Create(newDeployment)
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
		if err := ctx.controller.k8sClient.CoreV1().Services(ctx.gateway.Status.Resources.Service.Namespace).Delete(ctx.gateway.Status.Resources.Service.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if newService == nil {
		return nil, nil
	}

	if ctx.gateway.Status.Resources.Service == nil {
		ctx.updated = true
		return ctx.controller.k8sClient.CoreV1().Services(newService.Namespace).Create(newService)
	}

	currentMetadata := ctx.gateway.Status.Resources.Service
	currentService, err := ctx.controller.k8sClient.CoreV1().Services(currentMetadata.Namespace).Get(currentMetadata.Name, metav1.GetOptions{})
	if err != nil {
		ctx.updated = true
		return ctx.controller.k8sClient.CoreV1().Services(newService.Namespace).Create(newService)
	}

	if currentMetadata == nil {
		return nil, errors.New("service metadata is expected to be set in gateway object")
	}

	if currentService.Annotations != nil && currentService.Annotations[common.AnnotationResourceSpecHash] != newService.Annotations[common.AnnotationResourceSpecHash] {
		ctx.updated = true
		if err := ctx.controller.k8sClient.CoreV1().Services(currentMetadata.Namespace).Delete(currentMetadata.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		if ctx.gateway.Spec.Service != nil {
			return ctx.controller.k8sClient.CoreV1().Services(newService.Namespace).Create(newService)
		}
	}

	return currentService, nil
}
