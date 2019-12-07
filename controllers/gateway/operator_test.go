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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGatewayOperateLifecycle(t *testing.T) {
	controller := newController()
	ctx := newGatewayContext(gatewayObj.DeepCopy(), controller)
	gateway, err := controller.gatewayClient.ArgoprojV1alpha1().Gateways(gatewayObj.Namespace).Create(gatewayObj)
	assert.Nil(t, err)
	assert.NotNil(t, gateway)

	tests := []struct {
		name            string
		updateStateFunc func()
		testFunc        func(oldMetadata *v1alpha1.GatewayResource)
	}{
		{
			name:            "process a new gateway object",
			updateStateFunc: func() {},
			testFunc: func(oldMetadata *v1alpha1.GatewayResource) {
				assert.Nil(t, oldMetadata)
				deployment, err := controller.k8sClient.AppsV1().Deployments(ctx.gateway.Status.Resources.Deployment.Namespace).Get(ctx.gateway.Status.Resources.Deployment.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				service, err := controller.k8sClient.CoreV1().Services(ctx.gateway.Status.Resources.Service.Namespace).Get(ctx.gateway.Status.Resources.Service.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, service)
				assert.NotNil(t, ctx.gateway.Status.Resources)
				assert.Equal(t, ctx.gateway.Status.Resources.Deployment.Name, deployment.Name)
				assert.Equal(t, ctx.gateway.Status.Resources.Service.Name, service.Name)
				gateway, err := controller.gatewayClient.ArgoprojV1alpha1().Gateways(ctx.gateway.Namespace).Get(ctx.gateway.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, gateway)
				assert.Equal(t, gateway.Status.Phase, v1alpha1.NodePhaseRunning)
				ctx.gateway = gateway.DeepCopy()
			},
		},
		{
			name: "process a updated gateway object",
			updateStateFunc: func() {
				ctx.gateway.Spec.Template.Spec.Containers[0].Name = "new-name"
			},
			testFunc: func(oldMetadata *v1alpha1.GatewayResource) {
				currentMetadata := ctx.gateway.Status.Resources.DeepCopy()
				assert.NotEqual(t, oldMetadata.Deployment.Annotations[common.AnnotationResourceSpecHash], currentMetadata.Deployment.Annotations[common.AnnotationResourceSpecHash])
				assert.Equal(t, oldMetadata.Service.Annotations[common.AnnotationResourceSpecHash], currentMetadata.Service.Annotations[common.AnnotationResourceSpecHash])
			},
		},
		{
			name: "process a gateway object in error",
			updateStateFunc: func() {
				ctx.gateway.Status.Phase = v1alpha1.NodePhaseError
				ctx.gateway.Spec.Template.Spec.Containers[0].Name = "fixed-name"
			},
			testFunc: func(oldMetadata *v1alpha1.GatewayResource) {
				currentMetadata := ctx.gateway.Status.Resources.DeepCopy()
				assert.NotEqual(t, oldMetadata.Deployment.Annotations[common.AnnotationResourceSpecHash], currentMetadata.Deployment.Annotations[common.AnnotationResourceSpecHash])
				assert.Equal(t, oldMetadata.Service.Annotations[common.AnnotationResourceSpecHash], currentMetadata.Service.Annotations[common.AnnotationResourceSpecHash])
				gateway, err := controller.gatewayClient.ArgoprojV1alpha1().Gateways(ctx.gateway.Namespace).Get(ctx.gateway.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, gateway)
				assert.Equal(t, gateway.Status.Phase, v1alpha1.NodePhaseRunning)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldMetadata := ctx.gateway.Status.Resources.DeepCopy()
			test.updateStateFunc()
			err := ctx.operate()
			assert.Nil(t, err)
			test.testFunc(oldMetadata)
		})
	}
}

func TestPersistUpdates(t *testing.T) {
	controller := newController()
	ctx := newGatewayContext(gatewayObj.DeepCopy(), controller)
	gateway, err := controller.gatewayClient.ArgoprojV1alpha1().Gateways(gatewayObj.Namespace).Create(gatewayObj)
	assert.Nil(t, err)
	assert.NotNil(t, gateway)

	ctx.gateway = gateway
	ctx.gateway.Spec.Template.Name = "updated-name"
	gateway, err = PersistUpdates(controller.gatewayClient, ctx.gateway, controller.logger)
	assert.Nil(t, err)
	assert.Equal(t, gateway.Spec.Template.Name, "updated-name")
}

func TestReapplyUpdates(t *testing.T) {
	controller := newController()
	ctx := newGatewayContext(gatewayObj.DeepCopy(), controller)
	gateway, err := controller.gatewayClient.ArgoprojV1alpha1().Gateways(gatewayObj.Namespace).Create(gatewayObj)
	assert.Nil(t, err)
	assert.NotNil(t, gateway)

	ctx.gateway = gateway
	ctx.gateway.Spec.Template.Name = "updated-name"
	gateway, err = PersistUpdates(controller.gatewayClient, ctx.gateway, controller.logger)
	assert.Nil(t, err)
	assert.Equal(t, gateway.Spec.Template.Name, "updated-name")
}

func TestOperator_MarkPhase(t *testing.T) {
	controller := newController()
	ctx := newGatewayContext(gatewayObj.DeepCopy(), controller)
	assert.Equal(t, ctx.gateway.Status.Phase, v1alpha1.NodePhaseNew)
	assert.Equal(t, ctx.gateway.Status.Message, "")
	ctx.gateway.Status.Phase = v1alpha1.NodePhaseRunning
	ctx.markGatewayPhase(v1alpha1.NodePhaseRunning, "node is active")
	assert.Equal(t, ctx.gateway.Status.Phase, v1alpha1.NodePhaseRunning)
	assert.Equal(t, ctx.gateway.Status.Message, "node is active")
}

func TestOperator_UpdateGatewayState(t *testing.T) {
	controller := newController()
	ctx := newGatewayContext(gatewayObj.DeepCopy(), controller)
	gateway, err := controller.gatewayClient.ArgoprojV1alpha1().Gateways(gatewayObj.Namespace).Create(gatewayObj)
	assert.Nil(t, err)
	assert.NotNil(t, gateway)
	ctx.gateway = gateway.DeepCopy()
	assert.Equal(t, ctx.gateway.Status.Phase, v1alpha1.NodePhaseNew)
	ctx.gateway.Status.Phase = v1alpha1.NodePhaseRunning
	ctx.updated = true
	ctx.updateGatewayState()
	gateway, err = controller.gatewayClient.ArgoprojV1alpha1().Gateways(ctx.gateway.Namespace).Get(ctx.gateway.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, gateway)
	ctx.gateway = gateway.DeepCopy()
	assert.Equal(t, ctx.gateway.Status.Phase, v1alpha1.NodePhaseRunning)

	ctx.gateway.Status.Phase = v1alpha1.NodePhaseError
	ctx.updated = false
	ctx.updateGatewayState()
	gateway, err = controller.gatewayClient.ArgoprojV1alpha1().Gateways(ctx.gateway.Namespace).Get(ctx.gateway.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, gateway)
	ctx.gateway = gateway.DeepCopy()
	assert.Equal(t, ctx.gateway.Status.Phase, v1alpha1.NodePhaseRunning)
}
