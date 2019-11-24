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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildServiceResource builds a new service that exposes gateway.
func (opctx *operationContext) buildServiceResource() (*corev1.Service, error) {
	if opctx.gatewayObj.Spec.Service == nil {
		return nil, nil
	}
	service := opctx.gatewayObj.Spec.Service
	if service.Name == "" {
		service.Name = common.DefaultServiceName(opctx.gatewayObj.Name)
	}
	if service.Namespace == "" {
		service.Namespace = opctx.gatewayObj.Namespace
	}
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
	if deployment.Name == "" {
		deployment.Name = opctx.gatewayObj.Name
	}
	if deployment.Namespace == "" {
		deployment.Namespace = opctx.gatewayObj.Namespace
	}

	if err := controllerscommon.SetObjectMeta(opctx.gatewayObj, deployment, opctx.gatewayObj.GroupVersionKind()); err != nil {
		return nil, errors.Wrap(err, "failed to set the object metadata on the deployment object")
	}
	opctx.setupContainersForGatewayDeployment(deployment)
	return deployment, nil
}

// containers required for gateway deployment
func (opctx *operationContext) setupContainersForGatewayDeployment(deployment *appv1.Deployment) {
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
}

// createGatewayResources creates K8s resources corresponding to a gateway object
func (opctx *operationContext) createGatewayResources() error {
	// Gateway deployment has two components,
	// 1) Gateway Server   - Listen events from event source and dispatches the event to gateway client
	// 2) Gateway Client   - Listens for events from gateway server, convert them into cloudevents specification
	//                          compliant events and dispatch them to watchers.
	deployment, err := opctx.buildDeploymentResource()
	if err != nil {
		return err
	}

	if _, err := opctx.controller.k8sClient.AppsV1().Deployments(opctx.gatewayObj.Namespace).Create(deployment); err != nil {
		return err
	}

	opctx.logger.WithField(common.LabelDeploymentName, deployment.Name).Info("a deployment for the gateway is created")

	// expose gateway if service is configured
	if opctx.gatewayObj.Spec.Service != nil {
		svc, err := opctx.buildServiceResource()
		if err != nil {
			return err
		}

		if _, err := opctx.controller.k8sClient.CoreV1().Services(svc.Namespace).Create(svc); err != nil {
			return err
		}

		opctx.logger.WithField(common.LabelServiceName, svc.Name).Infoln("gateway service is created")
	}
	return nil
}

// updateGatewayResources updates the gateway deployment and service if the gateway resource spec has changed since last deployment
func (opctx *operationContext) updateGatewayResources() error {
	if err := Validate(opctx.gatewayObj); err != nil {
		opctx.logger.WithError(err).Error("gateway validation failed")

		if opctx.gatewayObj.Status.Phase != v1alpha1.NodePhaseError {
			opctx.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
		}

		return err
	}

	selector, err := controllerscommon.OwnerLabelSelector(opctx.gatewayObj.Name)
	if err != nil {
		return err
	}

	gatewayDeployment, err := opctx.buildDeploymentResource()
	if err != nil {
		return err
	}

	deployments, err := opctx.controller.k8sClient.AppsV1().Deployments(opctx.gatewayObj.Namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		if _, err := opctx.controller.k8sClient.AppsV1().Deployments(opctx.gatewayObj.Namespace).Create(gatewayDeployment); err != nil {
			return err
		}
	} else {
		deployment := deployments.Items[0]

		gatewayDeploymentHash, err := common.GetObjectHash(gatewayDeployment)
		if err != nil {
			return err
		}

		if deployment.Annotations != nil && deployment.Annotations[common.AnnotationResourceSpecHash] != gatewayDeploymentHash {
			if err := opctx.controller.k8sClient.AppsV1().Deployments(opctx.gatewayObj.Namespace).Delete(deployment.Name, &metav1.DeleteOptions{}); err != nil {
				return err
			}

			if _, err := opctx.controller.k8sClient.AppsV1().Deployments(opctx.gatewayObj.Namespace).Create(gatewayDeployment); err != nil {
				return err
			}
		}
	}

	if opctx.gatewayObj.Spec.Service != nil {
		serviceObj, err := opctx.buildServiceResource()
		if err != nil {
			return err
		}

		if _, err := opctx.controller.k8sClient.CoreV1().Services(opctx.gatewayObj.Namespace).Update(serviceObj); err != nil {
			return err
		}
	}

	opctx.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")

	return nil
}
