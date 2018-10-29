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
	fake_gw "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/selection"
	"testing"
)

func TestInstanceIDReq(t *testing.T) {
	controller := &GatewayController{
		Config: GatewayControllerConfig{
			InstanceID: "argo-events",
		},
	}

	req := controller.instanceIDReq()
	assert.Equal(t, common.LabelKeyGatewayControllerInstanceID, req.Key())

	req = controller.instanceIDReq()
	assert.Equal(t, selection.Equals, req.Operator())
	assert.True(t, req.Values().Has("argo-events"))
}

func TestNewGatewayInformer(t *testing.T) {
	controller := &GatewayController{
		Config: GatewayControllerConfig{
			Namespace:  "testing",
			InstanceID: "argo-events",
		},
		gatewayClientset: fake_gw.NewSimpleClientset(),
	}
	controller.newGatewayInformer()
}
