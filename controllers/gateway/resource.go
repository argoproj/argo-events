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
func (opctx *operationContext) buildServiceResource() (*corev1.Service, error) {
	if opctx.gatewayObj.Spec.Service == nil {
		return nil, nil
	}
	service := opctx.gatewayObj.Spec.Service.DeepCopy()
	if err := controllerscommon.SetObjectMeta(opctx.gatewayObj, service, opctx.gatewayObj.GroupVersionKind()); err != nil {
		return nil, err
	}
	return service, nil
}

// buildDeploymentResource builds a deployment resource for the gateway
func (opctx *operationContext) buildDeploymentResource() (*appv1.Deployment, error) {
	podTemplate := opctx.gatewayObj.Spec.Template.DeepCopy()

	replica := int32(opctx.gatewayObj.Spec.Replica)
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
	deployment.Spec.Template.Labels[common.LabelObjectName] = opctx.gatewayObj.Name

	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{},
		}
	}
	deployment.Spec.Selector.MatchLabels[common.LabelObjectName] = opctx.gatewayObj.Name

	opctx.setupContainersForGatewayDeployment(deployment)

	if err := controllerscommon.SetObjectMeta(opctx.gatewayObj, deployment, opctx.gatewayObj.GroupVersionKind()); err != nil {
		return nil, errors.Wrap(err, "failed to set the object metadata on the deployment object")
	}
	return deployment, nil
}

// containers required for gateway deployment
func (opctx *operationContext) setupContainersForGatewayDeployment(deployment *appv1.Deployment) *appv1.Deployment {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.EnvVarNamespace,
			Value: opctx.gatewayObj.Namespace,
		},
		{
			Name:  common.EnvVarEventSource,
			Value: opctx.gatewayObj.Spec.EventSourceRef.Name,
		},
		{
			Name:  common.EnvVarResourceName,
			Value: opctx.gatewayObj.Name,
		},
		{
			Name:  common.EnvVarControllerInstanceID,
			Value: opctx.controller.Config.InstanceID,
		},
		{
			Name:  common.EnvVarGatewayServerPort,
			Value: opctx.gatewayObj.Spec.ProcessorPort,
		},
	}

	for i, container := range deployment.Spec.Template.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		deployment.Spec.Template.Spec.Containers[i] = container
	}
	return deployment
}

// createGatewayResources creates gateway deployment and service
func (opctx *operationContext) createGatewayResources() error {
	if opctx.gatewayObj.Status.Resources == nil {
		opctx.gatewayObj.Status.Resources = &v1alpha1.GatewayResource{}
	}

	deployment, err := opctx.createGatewayDeployment()
	if err != nil {
		return err
	}
	opctx.gatewayObj.Status.Resources.Deployment = &deployment.ObjectMeta
	opctx.logger.WithField("name", deployment.Name).WithField("namespace", deployment.Namespace).Infoln("gateway deployment is created")

	if opctx.gatewayObj.Spec.Service != nil {
		service, err := opctx.createGatewayService()
		if err != nil {
			return err
		}
		opctx.gatewayObj.Status.Resources.Service = &service.ObjectMeta
		opctx.logger.WithField("name", service.Name).WithField("namespace", service.Namespace).Infoln("gateway service is created")
	}

	return nil
}

// createGatewayDeployment creates a deployment for the gateway
func (opctx *operationContext) createGatewayDeployment() (*appv1.Deployment, error) {
	deployment, err := opctx.buildDeploymentResource()
	if err != nil {
		return nil, err
	}
	return opctx.controller.k8sClient.AppsV1().Deployments(deployment.Namespace).Create(deployment)
}

// createGatewayService creates a service for the gateway
func (opctx *operationContext) createGatewayService() (*corev1.Service, error) {
	svc, err := opctx.buildServiceResource()
	if err != nil {
		return nil, err
	}
	return opctx.controller.k8sClient.CoreV1().Services(svc.Namespace).Create(svc)
}

// updateGatewayResources updates gateway deployment and service
func (opctx *operationContext) updateGatewayResources() error {
	deployment, err := opctx.updateGatewayDeployment()
	if err != nil {
		return err
	}
	if deployment != nil {
		opctx.gatewayObj.Status.Resources.Deployment = &deployment.ObjectMeta
		opctx.logger.WithField("name", deployment.Name).WithField("namespace", deployment.Namespace).Infoln("gateway deployment is updated")
	}

	service, err := opctx.updateGatewayService()
	if err != nil {
		return err
	}
	if service != nil {
		opctx.gatewayObj.Status.Resources.Service = &service.ObjectMeta
		opctx.logger.WithField("name", service.Name).WithField("namespace", service.Namespace).Infoln("gateway service is updated")
	}

	return nil
}

// updateGatewayDeployment updates the gateway deployment
func (opctx *operationContext) updateGatewayDeployment() (*appv1.Deployment, error) {
	newDeployment, err := opctx.buildDeploymentResource()
	if err != nil {
		return nil, err
	}

	oldDeploymentMetadata := opctx.gatewayObj.Status.Resources.Deployment
	if oldDeploymentMetadata == nil {
		return nil, errors.New("deployment metadata is expected to be set in gateway object")
	}

	oldDeployment, err := opctx.controller.k8sClient.AppsV1().Deployments(oldDeploymentMetadata.Namespace).Get(oldDeploymentMetadata.Name, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) {
			return opctx.controller.k8sClient.AppsV1().Deployments(newDeployment.Namespace).Create(newDeployment)
		}
		return nil, err
	}

	gatewayDeploymentHash, err := common.GetObjectHash(newDeployment)
	if err != nil {
		return nil, err
	}

	if oldDeployment.Annotations != nil && oldDeployment.Annotations[common.AnnotationResourceSpecHash] != gatewayDeploymentHash {
		if err := opctx.controller.k8sClient.AppsV1().Deployments(oldDeployment.Namespace).Delete(oldDeployment.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		return opctx.controller.k8sClient.AppsV1().Deployments(newDeployment.Namespace).Create(newDeployment)
	}

	return nil, nil
}

// updateGatewayService updates the gateway service
func (opctx *operationContext) updateGatewayService() (*corev1.Service, error) {
	serviceObj, err := opctx.buildServiceResource()
	if err != nil {
		return nil, err
	}
	if serviceObj == nil && opctx.gatewayObj.Status.Resources.Service != nil {
		if err := opctx.controller.k8sClient.CoreV1().Services(opctx.gatewayObj.Status.Resources.Service.Namespace).Delete(opctx.gatewayObj.Status.Resources.Service.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
	}

	if opctx.gatewayObj.Status.Resources.Service == nil {
		return opctx.controller.k8sClient.CoreV1().Services(serviceObj.Namespace).Create(serviceObj)
	}

	oldServiceMetadata := opctx.gatewayObj.Status.Resources.Service
	oldService, err := opctx.controller.k8sClient.CoreV1().Services(oldServiceMetadata.Namespace).Get(oldServiceMetadata.Name, metav1.GetOptions{})
	if err != nil {
		return opctx.controller.k8sClient.CoreV1().Services(serviceObj.Namespace).Create(serviceObj)
	}

	serviceHash, err := common.GetObjectHash(serviceObj)
	if oldService.Annotations != nil && oldService.Annotations[common.AnnotationResourceSpecHash] != serviceHash {
		if err := opctx.controller.k8sClient.CoreV1().Services(oldServiceMetadata.Namespace).Delete(oldServiceMetadata.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
		if opctx.gatewayObj.Spec.Service != nil {
			return opctx.controller.k8sClient.CoreV1().Services(serviceObj.Namespace).Create(serviceObj)
		}
	}

	return nil, nil
}
