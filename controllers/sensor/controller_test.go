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

package sensor

import (
	"fmt"
	"testing"
	"time"

	"github.com/argoproj/argo-events/common"
	fakesensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var (
	SensorControllerConfigmap  = common.DefaultConfigMapName("sensor-controller")
	SensorControllerInstanceID = "argo-events"
)

func getFakePodSharedIndexInformer(clientset kubernetes.Interface) cache.SharedIndexInformer {
	// NewListWatchFromClient doesn't work with fake client.
	// ref: https://github.com/kubernetes/client-go/issues/352
	return cache.NewSharedIndexInformer(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.CoreV1().Pods("").List(options)
		},
		WatchFunc: clientset.CoreV1().Pods("").Watch,
	}, &corev1.Pod{}, 1*time.Second, cache.Indexers{})
}

func getSensorController() *Controller {
	clientset := fake.NewSimpleClientset()
	done := make(chan struct{})
	informer := getFakePodSharedIndexInformer(clientset)
	go informer.Run(done)
	factory := informers.NewSharedInformerFactory(clientset, 0)
	podInformer := factory.Core().V1().Pods()
	go podInformer.Informer().Run(done)
	svcInformer := factory.Core().V1().Services()
	go svcInformer.Informer().Run(done)
	return &Controller{
		ConfigMap: SensorControllerConfigmap,
		Namespace: common.DefaultControllerNamespace,
		Config: ControllerConfig{
			Namespace:  common.DefaultControllerNamespace,
			InstanceID: SensorControllerInstanceID,
		},
		k8sClient:    clientset,
		sensorClient: fakesensor.NewSimpleClientset(),
		informer:     informer,
		queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		logger:       common.NewArgoEventsLogger(),
	}
}

func TestGatewayController(t *testing.T) {
	convey.Convey("Given a sensor controller, process queue items", t, func() {
		controller := getSensorController()

		convey.Convey("Create a resource queue, add new item and process it", func() {
			controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			controller.informer = controller.newSensorInformer()
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

	convey.Convey("Given a sensor controller, handle errors in queue", t, func() {
		controller := getSensorController()
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
