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
	"testing"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var gatewayObj = &v1alpha1.Gateway{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-gateway",
		Namespace: common.DefaultControllerNamespace,
	},
	Spec: v1alpha1.GatewaySpec{
		EventSourceRef: &v1alpha1.EventSourceRef{
			Name: "fake-event-source",
		},
		Replica:       1,
		Type:          apicommon.WebhookEvent,
		ProcessorPort: "8080",
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: "webhook-gateway",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "gateway-client",
						Image:           "argoproj/gateway-client",
						ImagePullPolicy: corev1.PullAlways,
					},
					{
						Name:            "gateway-server",
						ImagePullPolicy: corev1.PullAlways,
						Image:           "argoproj/webhook-gateway",
					},
				},
			},
		},
		Service: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "webhook-gateway-svc",
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
				Selector: map[string]string{
					"gateway-name": "webhook-gateway",
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "server-port",
						Port:       12000,
						TargetPort: intstr.FromInt(12000),
					},
				},
			},
		},
		EventProtocol: &apicommon.EventProtocol{
			Type: apicommon.HTTP,
			Http: apicommon.Http{
				Port: "9330",
			},
		},
		Subscribers: &v1alpha1.Subscribers{
			HTTP: []string{"http://fake-sensor.fake.svc.cluser.local:8080/"},
		},
	},
}

func TestResource_BuildServiceResource(t *testing.T) {
	controller := newController()
	opCtx := newGatewayContext(gatewayObj, controller)
	svc := opCtx.gateway.Spec.Service.DeepCopy()
	opCtx.gateway.Spec.Service = nil

	// If no service is defined
	service, err := opCtx.buildServiceResource()
	assert.Nil(t, err)
	assert.Nil(t, service)
	opCtx.gateway.Spec.Service = svc

	// If service is defined
	service, err = opCtx.buildServiceResource()
	assert.Nil(t, err)
	assert.NotNil(t, service)

	opCtx.gateway.Spec.Service.Name = ""
	opCtx.gateway.Spec.Service.Namespace = ""

	service, err = opCtx.buildServiceResource()
	assert.Nil(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, service.Name, opCtx.gateway.Name)
	assert.Equal(t, service.Namespace, opCtx.gateway.Namespace)

	newSvc, err := controller.k8sClient.CoreV1().Services(service.Namespace).Create(service)
	assert.Nil(t, err)
	assert.NotNil(t, newSvc)
	assert.Equal(t, newSvc.Name, opCtx.gateway.Name)
	assert.Equal(t, len(newSvc.Spec.Ports), 1)
	assert.Equal(t, newSvc.Spec.Type, corev1.ServiceTypeLoadBalancer)
}

func TestResource_BuildDeploymentResource(t *testing.T) {
	controller := newController()
	ctx := newGatewayContext(gatewayObj, controller)
	deployment, err := ctx.buildDeploymentResource()
	assert.Nil(t, err)
	assert.NotNil(t, deployment)

	for _, container := range deployment.Spec.Template.Spec.Containers {
		assert.NotNil(t, container.Env)
		assert.Equal(t, container.Env[0].Name, common.EnvVarNamespace)
		assert.Equal(t, container.Env[0].Value, ctx.gateway.Namespace)
		assert.Equal(t, container.Env[1].Name, common.EnvVarEventSource)
		assert.Equal(t, container.Env[1].Value, ctx.gateway.Spec.EventSourceRef.Name)
		assert.Equal(t, container.Env[2].Name, common.EnvVarResourceName)
		assert.Equal(t, container.Env[2].Value, ctx.gateway.Name)
		assert.Equal(t, container.Env[3].Name, common.EnvVarControllerInstanceID)
		assert.Equal(t, container.Env[3].Value, ctx.controller.Config.InstanceID)
		assert.Equal(t, container.Env[4].Name, common.EnvVarGatewayServerPort)
		assert.Equal(t, container.Env[4].Value, ctx.gateway.Spec.ProcessorPort)
	}

	newDeployment, err := controller.k8sClient.AppsV1().Deployments(deployment.Namespace).Create(deployment)
	assert.Nil(t, err)
	assert.NotNil(t, newDeployment)
	assert.Equal(t, newDeployment.Labels[common.LabelOwnerName], ctx.gateway.Name)
	assert.NotNil(t, newDeployment.Annotations[common.AnnotationResourceSpecHash])
}

func TestResource_CreateGatewayResource(t *testing.T) {
	tests := []struct {
		name       string
		updateFunc func(ctx *gatewayContext)
		testFunc   func(controller *Controller, ctx *gatewayContext, t *testing.T)
	}{
		{
			name:       "gateway with deployment and service",
			updateFunc: func(ctx *gatewayContext) {},
			testFunc: func(controller *Controller, ctx *gatewayContext, t *testing.T) {
				deploymentMetadata := ctx.gateway.Status.Resources.Deployment
				serviceMetadata := ctx.gateway.Status.Resources.Service
				deployment, err := controller.k8sClient.AppsV1().Deployments(deploymentMetadata.Namespace).Get(deploymentMetadata.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				service, err := controller.k8sClient.CoreV1().Services(serviceMetadata.Namespace).Get(serviceMetadata.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, service)
			},
		},
		{
			name: "gateway with zero deployment replica",
			updateFunc: func(ctx *gatewayContext) {
				ctx.gateway.Spec.Replica = 0
			},
			testFunc: func(controller *Controller, ctx *gatewayContext, t *testing.T) {
				deploymentMetadata := ctx.gateway.Status.Resources.Deployment
				deployment, err := controller.k8sClient.AppsV1().Deployments(deploymentMetadata.Namespace).Get(deploymentMetadata.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				assert.Equal(t, *deployment.Spec.Replicas, int32(1))
			},
		},
		{
			name: "gateway with empty service template",
			updateFunc: func(ctx *gatewayContext) {
				ctx.gateway.Spec.Service = nil
			},
			testFunc: func(controller *Controller, ctx *gatewayContext, t *testing.T) {
				deploymentMetadata := ctx.gateway.Status.Resources.Deployment
				deployment, err := controller.k8sClient.AppsV1().Deployments(deploymentMetadata.Namespace).Get(deploymentMetadata.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				assert.Nil(t, ctx.gateway.Status.Resources.Service)
			},
		},
		{
			name: "gateway with resources in different namespaces",
			updateFunc: func(ctx *gatewayContext) {
				ctx.gateway.Spec.Template.Namespace = "new-namespace"
				ctx.gateway.Spec.Service.Namespace = "new-namespace"
			},
			testFunc: func(controller *Controller, ctx *gatewayContext, t *testing.T) {
				deploymentMetadata := ctx.gateway.Status.Resources.Deployment
				serviceMetadata := ctx.gateway.Status.Resources.Service
				deployment, err := controller.k8sClient.AppsV1().Deployments(deploymentMetadata.Namespace).Get(deploymentMetadata.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				service, err := controller.k8sClient.CoreV1().Services(serviceMetadata.Namespace).Get(serviceMetadata.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, service)
				assert.NotEqual(t, ctx.gateway.Namespace, deployment.Namespace)
				assert.NotEqual(t, ctx.gateway.Namespace, service.Namespace)
			},
		},
		{
			name: "gateway with resources with empty names and namespaces",
			updateFunc: func(ctx *gatewayContext) {
				ctx.gateway.Spec.Template.Name = ""
				ctx.gateway.Spec.Service.Name = ""
			},
			testFunc: func(controller *Controller, ctx *gatewayContext, t *testing.T) {
				deploymentMetadata := ctx.gateway.Status.Resources.Deployment
				serviceMetadata := ctx.gateway.Status.Resources.Service
				deployment, err := controller.k8sClient.AppsV1().Deployments(deploymentMetadata.Namespace).Get(deploymentMetadata.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				service, err := controller.k8sClient.CoreV1().Services(serviceMetadata.Namespace).Get(serviceMetadata.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, ctx.gateway.Name, deployment.Name)
				assert.Equal(t, ctx.gateway.Name, service.Name)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newController()
			ctx := newGatewayContext(gatewayObj.DeepCopy(), controller)
			test.updateFunc(ctx)
			err := ctx.createGatewayResources()
			assert.Nil(t, err)
			test.testFunc(controller, ctx, t)
		})
	}
}

func TestResource_UpdateGatewayResource(t *testing.T) {
	controller := newController()
	ctx := newGatewayContext(gatewayObj.DeepCopy(), controller)
	err := ctx.createGatewayResources()
	assert.Nil(t, err)

	tests := []struct {
		name       string
		updateFunc func()
		testFunc   func(t *testing.T, oldMetadata *v1alpha1.GatewayResource)
	}{
		{
			name: "update deployment resource on gateway template change",
			updateFunc: func() {
				ctx.gateway.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
				ctx.gateway.Spec.Service.Spec.Type = corev1.ServiceTypeNodePort
			},
			testFunc: func(t *testing.T, oldMetadata *v1alpha1.GatewayResource) {
				currentMetadata := ctx.gateway.Status.Resources
				deployment, err := controller.k8sClient.AppsV1().Deployments(currentMetadata.Deployment.Namespace).Get(currentMetadata.Deployment.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				assert.NotEqual(t, deployment.Annotations[common.AnnotationResourceSpecHash], oldMetadata.Deployment.Annotations[common.AnnotationResourceSpecHash])
				service, err := controller.k8sClient.CoreV1().Services(currentMetadata.Service.Namespace).Get(currentMetadata.Service.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, service)
				assert.NotEqual(t, service.Annotations[common.AnnotationResourceSpecHash], oldMetadata.Service.Annotations[common.AnnotationResourceSpecHash])
			},
		},
		{
			name: "delete service resource if gateway service spec is removed",
			updateFunc: func() {
				ctx.gateway.Spec.Service = nil
			},
			testFunc: func(t *testing.T, oldMetadata *v1alpha1.GatewayResource) {
				currentMetadata := ctx.gateway.Status.Resources
				deployment, err := controller.k8sClient.AppsV1().Deployments(currentMetadata.Deployment.Namespace).Get(currentMetadata.Deployment.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				assert.Equal(t, deployment.Annotations[common.AnnotationResourceSpecHash], oldMetadata.Deployment.Annotations[common.AnnotationResourceSpecHash])
				assert.Nil(t, ctx.gateway.Status.Resources.Service)
				service, err := controller.k8sClient.CoreV1().Services(oldMetadata.Service.Namespace).Get(oldMetadata.Service.Name, metav1.GetOptions{})
				assert.NotNil(t, err)
				assert.Equal(t, apierror.IsNotFound(err), true)
				assert.Nil(t, service)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata := ctx.gateway.Status.Resources.DeepCopy()
			test.updateFunc()
			err := ctx.updateGatewayResources()
			assert.Nil(t, err)
			test.testFunc(t, metadata)
		})
	}
}
