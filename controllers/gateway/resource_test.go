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
		Template: v1alpha1.Template{
			ServiceAccountName: "fake-sa",
			Container: &corev1.Container{
				Image: "argoproj/fake-image",
			},
		},
		Service: &v1alpha1.Service{
			Ports: []corev1.ServicePort{
				{
					Name:       "server-port",
					Port:       12000,
					TargetPort: intstr.FromInt(12000),
				},
			},
		},
		Subscribers: &v1alpha1.Subscribers{
			HTTP: []string{"http://fake-sensor.fake.svc.cluser.local:8080/"},
		},
	},
}

var gatewayObjNoTemplate = &v1alpha1.Gateway{
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
		Service: &v1alpha1.Service{
			Ports: []corev1.ServicePort{
				{
					Name:       "server-port",
					Port:       12000,
					TargetPort: intstr.FromInt(12000),
				},
			},
		},
		Subscribers: &v1alpha1.Subscribers{
			HTTP: []string{"http://fake-sensor.fake.svc.cluser.local:8080/"},
		},
	},
}

func TestResource_BuildServiceResource(t *testing.T) {
	gwObjs := []*v1alpha1.Gateway{gatewayObj, gatewayObjNoTemplate}
	for _, gwObj := range gwObjs {
		controller := newController()
		opCtx := newGatewayContext(gwObj, controller)

		service, err := opCtx.buildServiceResource()
		assert.Nil(t, err)
		assert.NotNil(t, service)
		assert.Equal(t, service.Name, gatewayObj.Name+"-gateway")
		assert.Equal(t, service.Namespace, opCtx.gateway.Namespace)

		newSvc, err := controller.k8sClient.CoreV1().Services(service.Namespace).Create(service)
		assert.Nil(t, err)
		assert.NotNil(t, newSvc)
		assert.Equal(t, newSvc.Name, service.Name)
		assert.Equal(t, len(newSvc.Spec.Ports), 1)
		assert.Equal(t, newSvc.Spec.Type, corev1.ServiceTypeClusterIP)
	}
}

func TestResource_BuildDeploymentResource(t *testing.T) {
	gwObjs := []*v1alpha1.Gateway{gatewayObj, gatewayObjNoTemplate}
	for _, gwObj := range gwObjs {
		controller := newController()
		ctx := newGatewayContext(gwObj, controller)
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
}
