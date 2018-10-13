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
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"

	fakesensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
)

// fakeController is a wrapper around the sensorController to allow efficient test setup/cleanup
type fakeController struct {
	*SensorController
}

func (f *fakeController) setup(namespace string) {
	f.SensorController = &SensorController{
		ConfigMap:   "configmap",
		Namespace: namespace,
		Config: SensorControllerConfig{
			Namespace: namespace,
		},
		kubeClientset:   fake.NewSimpleClientset(),
		sensorClientset: fakesensor.NewSimpleClientset(),
		queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func (f *fakeController) teardown() {
	f.queue.ShutDown()
}

func newFakeController() *fakeController {
	fakeController := &fakeController{}
	fakeController.setup(v1.NamespaceDefault)
	return fakeController
}

func TestProcessNextItem(t *testing.T) {
	controller := newFakeController()
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
	controller := newFakeController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	controller.queue.Add("hi")
	controller.handleErr(nil, "hi")

	controller.queue.Add("bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
}
