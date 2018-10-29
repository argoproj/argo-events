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

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-events/common"
	fakesensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
)

var (
	SensorControllerConfigmap  = common.DefaultConfigMapName("sensor-controller")
	SensorControllerInstanceID = "argo-events"
)

func sensorController() *SensorController {
	return &SensorController{
		ConfigMap: SensorControllerConfigmap,
		Namespace: common.DefaultControllerNamespace,
		Config: SensorControllerConfig{
			Namespace:  common.DefaultControllerNamespace,
			InstanceID: SensorControllerInstanceID,
		},
		kubeClientset:   fake.NewSimpleClientset(),
		sensorClientset: fakesensor.NewSimpleClientset(),
		queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func TestProcessNextItem(t *testing.T) {
	controller := sensorController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	controller.informer = controller.newSensorInformer()

	controller.queue.Add("hi")
	res := controller.processNextItem()
	assert.True(t, res)

	controller.queue.ShutDown()
	res = controller.processNextItem()
	assert.False(t, res)
}

func TestHandleErr(t *testing.T) {
	controller := sensorController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	controller.queue.Add("hi")
	controller.handleErr(nil, "hi")

	controller.queue.Add("bye")
	err := controller.handleErr(fmt.Errorf("real error"), "bye")
	assert.Nil(t, err)
}
