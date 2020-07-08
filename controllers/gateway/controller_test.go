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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	fakeeventbus "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/fake"
	fakegateway "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
)

func newController() *Controller {
	controller := &Controller{
		ConfigMap: configmapName,
		Namespace: common.DefaultControllerNamespace,
		Config: ControllerConfig{
			Namespace:  common.DefaultControllerNamespace,
			InstanceID: "argo-events",
		},
		clientImage:    "argoproj/gateway-client",
		serverImage:    "argoproj/gateway-server",
		k8sClient:      fake.NewSimpleClientset(),
		gatewayClient:  fakegateway.NewSimpleClientset(),
		eventBusClient: fakeeventbus.NewSimpleClientset(),
		queue:          workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		logger:         common.NewArgoEventsLogger(),
	}
	informer, err := controller.newGatewayInformer()
	if err != nil {
		panic(err)
	}
	controller.informer = informer
	return controller
}

func TestGatewayController_ProcessNextItem(t *testing.T) {
	controller := newController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	gw := &v1alpha1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-gateway",
			Namespace: common.DefaultControllerNamespace,
		},
		Spec: v1alpha1.GatewaySpec{},
	}
	err := controller.informer.GetIndexer().Add(gw)
	assert.Nil(t, err)

	controller.queue.Add("fake-gateway")
	res := controller.processNextItem()
	assert.Equal(t, res, true)

	controller.queue.ShutDown()
	res = controller.processNextItem()
	assert.Equal(t, res, false)
}

func TestGatewayController_HandleErr(t *testing.T) {
	controller := newController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	controller.queue.Add("hi")
	var err error
	for i := 0; i < 21; i++ {
		err = controller.handleErr(fmt.Errorf("real error"), "bye")
	}
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "exceeded max requeues")
}
