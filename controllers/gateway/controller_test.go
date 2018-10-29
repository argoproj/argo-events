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
	fakegateway "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
	"testing"
	"github.com/argoproj/argo-events/common"
)

func getGatewayController() *GatewayController {
	return &GatewayController{
		ConfigMap:   configmapName,
		Namespace: common.DefaultControllerNamespace,
		Config: GatewayControllerConfig{
			Namespace: common.DefaultControllerNamespace,
		},
		kubeClientset:    fake.NewSimpleClientset(),
		gatewayClientset: fakegateway.NewSimpleClientset(),
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func TestProcessNextItem(t *testing.T) {
	controller := getGatewayController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	controller.informer = controller.newGatewayInformer()

	controller.queue.Add("hi")
	res := controller.processNextItem()
	assert.True(t, res)

	controller.queue.ShutDown()
	res = controller.processNextItem()
	assert.False(t, res)
}

func TestHandleErr(t *testing.T) {
	controller := getGatewayController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	controller.queue.Add("hi")
	controller.handleErr(nil, "hi")

	controller.queue.Add("bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
}
