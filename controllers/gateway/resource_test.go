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
	assert.Equal(t, service.Name, common.DefaultServiceName(opCtx.gateway.Name))
	assert.Equal(t, service.Namespace, opCtx.gateway.Namespace)

	newSvc, err := controller.k8sClient.CoreV1().Services(service.Namespace).Create(service)
	assert.Nil(t, err)
	assert.NotNil(t, newSvc)
	assert.Equal(t, newSvc.Name, opCtx.gateway.Spec.Service.Name)
	assert.Equal(t, len(newSvc.Spec.Ports), 1)
	assert.Equal(t, newSvc.Spec.Type, corev1.ServiceTypeLoadBalancer)
}

func TestResource_BuildDeploymentResource(t *testing.T) {
	controller := newController()
	opCtx := newGatewayContext(gatewayObj, controller)
	deployment, err := opCtx.buildDeploymentResource()
	assert.Nil(t, err)
	assert.NotNil(t, deployment)

	newDeployment, err := controller.k8sClient.AppsV1().Deployments(deployment.Namespace).Create(deployment)
	assert.Nil(t, err)
	assert.NotNil(t, newDeployment)
	assert.Equal(t, newDeployment.Labels[common.LabelOwnerName], opCtx.gateway.Name)
	assert.NotNil(t, newDeployment.Annotations[common.AnnotationResourceSpecHash])
}

func TestResource_SetupContainersForGatewayDeployment(t *testing.T) {
	controller := newController()
	opctx := newGatewayContext(gatewayObj, controller)
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
		assert.Equal(t, container.Env[0].Value, opctx.gateway.Namespace)
		assert.Equal(t, container.Env[1].Name, common.EnvVarEventSource)
		assert.Equal(t, container.Env[1].Value, opctx.gateway.Spec.EventSourceRef.Name)
		assert.Equal(t, container.Env[2].Name, common.EnvVarResourceName)
		assert.Equal(t, container.Env[2].Value, opctx.gateway.Name)
		assert.Equal(t, container.Env[3].Name, common.EnvVarControllerInstanceID)
		assert.Equal(t, container.Env[3].Value, opctx.controller.Config.InstanceID)
		assert.Equal(t, container.Env[4].Name, common.EnvVarGatewayServerPort)
		assert.Equal(t, container.Env[4].Value, opctx.gateway.Spec.ProcessorPort)
	}
}

func TestResource_CreateGatewayResource(t *testing.T) {
	tests := []struct {
		name               string
		gatewayObj         *v1alpha1.Gateway
		deploymentMetadata *metav1.ObjectMeta
		deploymentNotFound bool
		serviceMetadata    *metav1.ObjectMeta
		serviceNotFound    bool
	}{
		{
			name:       "gateway with deployment and service",
			gatewayObj: gatewayObj.DeepCopy(),
			deploymentMetadata: &metav1.ObjectMeta{
				Name:      gatewayObj.Spec.Template.Name,
				Namespace: gatewayObj.Namespace,
			},
			serviceMetadata: &metav1.ObjectMeta{
				Name:      gatewayObj.Spec.Service.Name,
				Namespace: gatewayObj.Namespace,
			},
			deploymentNotFound: false,
			serviceNotFound:    false,
		},
		{
			name: "gateway with no service template",
			gatewayObj: &v1alpha1.Gateway{
				ObjectMeta: gatewayObj.ObjectMeta,
				Spec: v1alpha1.GatewaySpec{
					Template:       gatewayObj.Spec.Template,
					EventSourceRef: gatewayObj.Spec.EventSourceRef,
					Type:           gatewayObj.Spec.Type,
					Service:        nil,
					ProcessorPort:  gatewayObj.Spec.ProcessorPort,
					EventProtocol:  gatewayObj.Spec.EventProtocol,
					Replica:        0,
				},
			},
			deploymentMetadata: &metav1.ObjectMeta{
				Name:      gatewayObj.Spec.Template.Name,
				Namespace: gatewayObj.Namespace,
			},
			serviceMetadata: &metav1.ObjectMeta{
				Name:      gatewayObj.Spec.Service.Name,
				Namespace: gatewayObj.Namespace,
			},
			serviceNotFound:    true,
			deploymentNotFound: false,
		},
		{
			name: "gateway with resources in different namespaces",
			gatewayObj: &v1alpha1.Gateway{
				ObjectMeta: gatewayObj.ObjectMeta,
				Spec: v1alpha1.GatewaySpec{
					Template: &corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      gatewayObj.Spec.Template.Name,
							Namespace: "new-namespace",
						},
						Spec: gatewayObj.Spec.Template.Spec,
					},
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      gatewayObj.Spec.Service.Name,
							Namespace: "new-namespace",
						},
						Spec: gatewayObj.Spec.Service.Spec,
					},
					EventSourceRef: gatewayObj.Spec.EventSourceRef,
					Type:           gatewayObj.Spec.Type,
					ProcessorPort:  gatewayObj.Spec.ProcessorPort,
					EventProtocol:  gatewayObj.Spec.EventProtocol,
					Replica:        0,
				},
			},
			deploymentMetadata: &metav1.ObjectMeta{
				Name:      gatewayObj.Spec.Template.Name,
				Namespace: "new-namespace",
			},
			serviceMetadata: &metav1.ObjectMeta{
				Name:      gatewayObj.Spec.Service.Name,
				Namespace: "new-namespace",
			},
			serviceNotFound:    false,
			deploymentNotFound: false,
		},
		{
			name: "gateway with resources with empty names and namespaces",
			gatewayObj: &v1alpha1.Gateway{
				ObjectMeta: gatewayObj.ObjectMeta,
				Spec: v1alpha1.GatewaySpec{
					Template: &corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "",
							Namespace: "",
						},
						Spec: gatewayObj.Spec.Template.Spec,
					},
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "",
							Namespace: "",
						},
						Spec: gatewayObj.Spec.Service.Spec,
					},
					EventSourceRef: gatewayObj.Spec.EventSourceRef,
					Type:           gatewayObj.Spec.Type,
					ProcessorPort:  gatewayObj.Spec.ProcessorPort,
					EventProtocol:  gatewayObj.Spec.EventProtocol,
					Replica:        0,
				},
			},
			deploymentMetadata: &metav1.ObjectMeta{
				Name:      gatewayObj.Name,
				Namespace: gatewayObj.Namespace,
			},
			serviceMetadata: &metav1.ObjectMeta{
				Name:      gatewayObj.Name,
				Namespace: gatewayObj.Namespace,
			},
			serviceNotFound:    false,
			deploymentNotFound: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newController()
			opctx := newGatewayContext(test.gatewayObj, controller)
			err := opctx.createGatewayResources()
			assert.Nil(t, err)
			deployment, err := controller.k8sClient.AppsV1().Deployments(test.deploymentMetadata.Namespace).Get(test.deploymentMetadata.Name, metav1.GetOptions{})
			assert.Equal(t, apierror.IsNotFound(err), test.deploymentNotFound)
			service, err := controller.k8sClient.CoreV1().Services(test.serviceMetadata.Namespace).Get(test.serviceMetadata.Name, metav1.GetOptions{})
			assert.Equal(t, apierror.IsNotFound(err), test.serviceNotFound)

			assert.NotNil(t, test.gatewayObj.Status.Resources)

			if !test.deploymentNotFound {
				assert.NotNil(t, deployment)
				assert.NotNil(t, deployment.Labels)
				assert.GreaterOrEqual(t, int(*deployment.Spec.Replicas), 1)
				assert.NotNil(t, deployment.Annotations)
				assert.Equal(t, deployment.Labels[common.LabelOwnerName], opctx.gateway.Name)
				deploymentObj, err := opctx.buildDeploymentResource()
				assert.Nil(t, err)
				deploymentObj.Annotations = nil
				deploymentHash, err := common.GetObjectHash(deploymentObj)
				assert.Nil(t, err)
				assert.Equal(t, deployment.Annotations[common.AnnotationResourceSpecHash], deploymentHash)
			}

			if !test.serviceNotFound {
				assert.NotNil(t, service)
				assert.NotNil(t, service.Labels)
				assert.NotNil(t, service.Annotations)
				assert.Equal(t, service.Labels[common.LabelOwnerName], opctx.gateway.Name)
				serviceObj, err := opctx.buildServiceResource()
				assert.Nil(t, err)
				serviceObj.Annotations = nil
				serviceHash, err := common.GetObjectHash(serviceObj)
				assert.Nil(t, err)
				assert.Equal(t, service.Annotations[common.AnnotationResourceSpecHash], serviceHash)
			}
		})
	}
}

func TestResource_UpdateGatewayResource(t *testing.T) {
	controller := newController()
	opctx := newGatewayContext(gatewayObj.DeepCopy(), controller)
	err := opctx.createGatewayResources()
	assert.Nil(t, err)

	oldDeploymentMetadata := opctx.gateway.Status.Resources.Deployment
	oldServiceMetadata := opctx.gateway.Status.Resources.Service

	tests := []struct {
		updateFunc      func()
		name            string
		serviceNotFound bool
	}{
		{
			name: "update deployment resource on gateway template change",
			updateFunc: func() {
				opctx.gateway.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
				opctx.gateway.Spec.Service.Spec.Type = corev1.ServiceTypeClusterIP
			},
			serviceNotFound: false,
		},
		{
			name: "delete service resource if gateway service spec is removed",
			updateFunc: func() {
				opctx.gateway.Spec.Service = nil
			},
			serviceNotFound: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.updateFunc()
			err := opctx.updateGatewayResources()
			assert.Nil(t, err)
			deployment, err := controller.k8sClient.AppsV1().Deployments(opctx.gateway.Namespace).Get(opctx.gateway.Spec.Template.Name, metav1.GetOptions{})
			assert.Nil(t, err)
			assert.NotNil(t, deployment)
			assert.NotEqual(t, deployment.Annotations[common.AnnotationResourceSpecHash], oldDeploymentMetadata.Annotations[common.AnnotationResourceSpecHash])
			if !test.serviceNotFound {
				service, err := controller.k8sClient.CoreV1().Services(opctx.gateway.Namespace).Get(opctx.gateway.Spec.Service.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotEqual(t, service.Annotations[common.AnnotationResourceSpecHash], oldServiceMetadata.Annotations[common.AnnotationResourceSpecHash])
			}
		})
	}
}
