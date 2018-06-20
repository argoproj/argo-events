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

package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"

	fakesensor "github.com/argoproj/argo-events/pkg/client/clientset/versioned/fake"
)

// fakeController is a wrapper around the sensorController to allow efficient test setup/cleanup
type fakeController struct {
	*SensorController
}

func (f *fakeController) setup(namespace string) {
	f.SensorController = NewSensorController(nil, fake.NewSimpleClientset(), fakesensor.NewSimpleClientset(), zap.NewNop().Sugar(), "configmap")
	f.Config = SensorControllerConfig{Namespace: namespace}
}

func (f *fakeController) teardown() {
	f.ssQueue.ShutDown()
	f.podQueue.ShutDown()
}

func newFakeController() *fakeController {
	fakeController := &fakeController{}
	fakeController.setup(v1.NamespaceDefault)
	return fakeController
}

func TestProcessNextJob(t *testing.T) {
	controller := newFakeController()
	controller.podQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	controller.podInformer = controller.newPodInformer()

	controller.podQueue.Add("hi")
	res := controller.processNextPod()
	assert.True(t, res)

	controller.podQueue.ShutDown()
	res = controller.processNextPod()
	assert.False(t, res)
}

func TestProcessNextItem(t *testing.T) {
	controller := newFakeController()
	controller.ssQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	controller.ssInformer = controller.newSensorInformer()

	controller.ssQueue.Add("hi")
	res := controller.processNextItem()
	assert.True(t, res)

	controller.ssQueue.ShutDown()
	res = controller.processNextItem()
	assert.False(t, res)
}

func TestHandleErr(t *testing.T) {
	controller := newFakeController()
	controller.ssQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	controller.ssQueue.Add("hi")
	controller.handleErr(nil, "hi")

	controller.ssQueue.Add("bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
}
