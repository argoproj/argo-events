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
	"github.com/stretchr/testify/assert"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	gatewayPodName = "webhook-gateway"
	gatewaySvcName = "webhook-gateway-svc"
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
		testFunc        func()
	}{
		{
			name:            "process a new gateway object",
			updateStateFunc: func() {},
			testFunc: func() {
				deployment, err := controller.k8sClient.AppsV1().Deployments(ctx.gateway.Status.Resources.Deployment.Namespace).Get(ctx.gateway.Status.Resources.Deployment.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.Nil(t, deployment)
				service, err := controller.k8sClient.CoreV1().Services(ctx.gateway.Status.Resources.Service.Namespace).Get(ctx.gateway.Status.Resources.Service.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.Nil(t, service)
			},
		},
		{
			name: "process a gateway object",
			updateStateFunc: func() {

			},
			testFunc: func() {

			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.updateStateFunc()
			err := ctx.operate()
			assert.Nil(t, err)
			test.testFunc()
		})
	}
}
