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

	"github.com/argoproj/argo-events/common"
	fakegateway "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/smartystreets/goconvey/convey"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
)

func getGatewayController() *GatewayController {
	return &GatewayController{
		ConfigMap: configmapName,
		Namespace: common.DefaultControllerNamespace,
		Config: GatewayControllerConfig{
			Namespace:  common.DefaultControllerNamespace,
			InstanceID: "argo-events",
		},
		kubeClientset:    fake.NewSimpleClientset(),
		gatewayClientset: fakegateway.NewSimpleClientset(),
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func TestGatewayController(t *testing.T) {
	convey.Convey("Given a gateway controller, process queue items", t, func() {
		controller := getGatewayController()

		convey.Convey("Create a resource queue, add new item and process it", func() {
			controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			controller.informer = controller.newGatewayInformer()
			controller.queue.Add("hi")
			res := controller.processNextItem()

			convey.Convey("Item from queue must be successfully processed", func() {
				convey.So(res, convey.ShouldBeTrue)
			})

			convey.Convey("Shutdown queue and make sure queue does not process next item", func() {
				controller.queue.ShutDown()
				res := controller.processNextItem()
				convey.So(res, convey.ShouldBeFalse)
			})
		})
	})

	convey.Convey("Given a gateway controller, handle errors in queue", t, func() {
		controller := getGatewayController()
		convey.Convey("Create a resource queue and add an item", func() {
			controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			controller.queue.Add("hi")
			convey.Convey("Handle an nil error", func() {
				err := controller.handleErr(nil, "hi")
				convey.So(err, convey.ShouldBeNil)
			})
			convey.Convey("Exceed max requeues", func() {
				controller.queue.Add("bye")
				var err error
				for i := 0; i < 21; i++ {
					err = controller.handleErr(fmt.Errorf("real error"), "bye")
				}
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldEqual, "exceeded max requeues")
			})
		})
	})
}
