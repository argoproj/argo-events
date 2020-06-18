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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	fakesensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
)

var (
	SensorControllerConfigmap  = "sensor-controller-configmap"
	SensorControllerInstanceID = "argo-events"
)

func getController() *Controller {
	clientset := fake.NewSimpleClientset()
	controller := &Controller{
		ConfigMap: SensorControllerConfigmap,
		Namespace: common.DefaultControllerNamespace,
		Config: ControllerConfig{
			Namespace:  common.DefaultControllerNamespace,
			InstanceID: SensorControllerInstanceID,
		},
		TemplateSpec: &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "sensor",
						Image:           "argoproj/sensor",
						ImagePullPolicy: corev1.PullAlways,
					},
				},
				ServiceAccountName: "fake-sa",
			},
		},
		sensorImage:  "sensor-image",
		k8sClient:    clientset,
		sensorClient: fakesensor.NewSimpleClientset(),
		queue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		logger:       common.NewArgoEventsLogger(),
	}
	informer, err := controller.newSensorInformer()
	if err != nil {
		panic(err)
	}
	controller.informer = informer
	return controller
}

func TestController_ProcessNextItem(t *testing.T) {
	controller := getController()
	err := controller.informer.GetIndexer().Add(&v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-sensor",
			Namespace: "fake-namespace",
		},
		Spec: v1alpha1.SensorSpec{},
	})
	assert.Nil(t, err)
	controller.queue.Add("fake-sensor")
	res := controller.processNextItem()
	assert.Equal(t, res, true)
	controller.queue.ShutDown()
	res = controller.processNextItem()
	assert.Equal(t, res, false)
}

func TestController_HandleErr(t *testing.T) {
	controller := getController()
	controller.queue.Add("hi")
	err := controller.handleErr(nil, "hi")
	assert.Nil(t, err)
	controller.queue.Add("bye")
	for i := 0; i < 21; i++ {
		err = controller.handleErr(fmt.Errorf("real error"), "bye")
	}
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "exceeded max re-queues")
}
