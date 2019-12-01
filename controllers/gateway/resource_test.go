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
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
		Watchers: &v1alpha1.NotificationWatchers{
			Sensors: []v1alpha1.SensorNotificationWatcher{
				{
					Name:      "fake-sensor",
					Namespace: common.DefaultControllerNamespace,
				},
			},
		},
	},
}

func TestResource_BuildServiceResource(t *testing.T) {
	controller := newController()
	opCtx := newOperationCtx(gatewayObj, controller)
	svc := opCtx.gatewayObj.Spec.Service.DeepCopy()
	opCtx.gatewayObj.Spec.Service = nil

	// If no service is defined
	service, err := opCtx.buildServiceResource()
	assert.Nil(t, err)
	assert.Nil(t, service)
	opCtx.gatewayObj.Spec.Service = svc

	// If service is defined
	service, err = opCtx.buildServiceResource()
	assert.Nil(t, err)
	assert.NotNil(t, service)

	opCtx.gatewayObj.Spec.Service.Name = ""
	opCtx.gatewayObj.Spec.Service.Namespace = ""

	service, err = opCtx.buildServiceResource()
	assert.Nil(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, service.Name, common.DefaultServiceName(opCtx.gatewayObj.Name))
	assert.Equal(t, service.Namespace, opCtx.gatewayObj.Namespace)

	newSvc, err := controller.k8sClient.CoreV1().Services(service.Namespace).Create(service)
	assert.Nil(t, err)
	assert.NotNil(t, newSvc)
	assert.Equal(t, newSvc.Name, opCtx.gatewayObj.Spec.Service.Name)
	assert.Equal(t, len(newSvc.Spec.Ports), 1)
	assert.Equal(t, newSvc.Spec.Type, corev1.ServiceTypeLoadBalancer)
}

func TestResource_BuildDeploymentResource(t *testing.T) {
	controller := newController()
	opCtx := newOperationCtx(gatewayObj, controller)
	deployment, err := opCtx.buildDeploymentResource()
	assert.Nil(t, err)
	assert.NotNil(t, deployment)

	newDeployment, err := controller.k8sClient.AppsV1().Deployments(deployment.Namespace).Create(deployment)
	assert.Nil(t, err)
	assert.NotNil(t, newDeployment)
	assert.Equal(t, newDeployment.Labels[common.LabelOwnerName], opCtx.gatewayObj.Name)
	assert.NotNil(t, newDeployment.Annotations[common.AnnotationResourceSpecHash])
}

func TestResource_SetupContainersForGatewayDeployment(t *testing.T) {
	controller := newController()
	opctx := newOperationCtx(gatewayObj, controller)
	deployment := opctx.setupContainersForGatewayDeployment(&appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-deployment",
		},
		Spec: appv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "fake-container-1",
						},
						{
							Name: "fake-container-2",
						},
					},
				},
			},
		},
	})
	assert.NotNil(t, deployment)
	for _, container := range deployment.Spec.Template.Spec.Containers {
		assert.NotNil(t, container.Env)
		assert.Equal(t, container.Env[0].Name, common.EnvVarNamespace)
		assert.Equal(t, container.Env[0].Value, opctx.gatewayObj.Namespace)
		assert.Equal(t, container.Env[1].Name, common.EnvVarEventSource)
		assert.Equal(t, container.Env[1].Value, opctx.gatewayObj.Spec.EventSourceRef.Name)
		assert.Equal(t, container.Env[2].Name, common.EnvVarResourceName)
		assert.Equal(t, container.Env[2].Value, opctx.gatewayObj.Name)
		assert.Equal(t, container.Env[3].Name, common.EnvVarControllerInstanceID)
		assert.Equal(t, container.Env[3].Value, opctx.controller.Config.InstanceID)
		assert.Equal(t, container.Env[4].Name, common.EnvVarGatewayServerPort)
		assert.Equal(t, container.Env[4].Value, opctx.gatewayObj.Spec.ProcessorPort)
	}
}

func TestResource_CreateGatewayResource(t *testing.T) {
	controller := newController()
	opctx := newOperationCtx(gatewayObj, controller)
	err := opctx.createGatewayResources()
	assert.Nil(t, err)
	deployment, err := controller.k8sClient.AppsV1().Deployments(opctx.gatewayObj.Namespace).Get(opctx.gatewayObj.Spec.Template.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, deployment)
	assert.NotNil(t, deployment.Labels)
	assert.NotNil(t, deployment.Annotations)
	assert.Equal(t, deployment.Labels[common.LabelOwnerName], opctx.gatewayObj.Name)
	deploymentObj, err := opctx.buildDeploymentResource()
	assert.Nil(t, err)
	deploymentObj.Annotations = nil
	deploymentHash, err := common.GetObjectHash(deploymentObj)
	assert.Nil(t, err)
	assert.Equal(t, deployment.Annotations[common.AnnotationResourceSpecHash], deploymentHash)
	service, err := controller.k8sClient.CoreV1().Services(opctx.gatewayObj.Namespace).Get(opctx.gatewayObj.Spec.Service.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, service.Spec.Type, corev1.ServiceTypeLoadBalancer)
}

func TestResource_UpdateGatewayResource(t *testing.T) {
	controller := newController()
	opctx := newOperationCtx(gatewayObj, controller)
	// Update gateway resources without the ones already deployed
	err := opctx.updateGatewayResources()
	assert.NotNil(t, err)
	deployment, err := controller.k8sClient.AppsV1().Deployments(opctx.gatewayObj.Namespace).Get(opctx.gatewayObj.Spec.Template.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, deployment)
	service, err := controller.k8sClient.CoreV1().Services(opctx.gatewayObj.Namespace).Get(opctx.gatewayObj.Spec.Service.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, service)
}
