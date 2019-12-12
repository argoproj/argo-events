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

package sensors

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// resyncs the sensor object for status updates
func (sensorCtx *sensorContext) syncSensor(ctx context.Context) cache.Controller {
	sensorCtx.logger.Info("watching sensor updates")
	source := sensorCtx.newSensorWatch()
	_, controller := cache.NewInformer(
		source,
		&v1alpha1.Sensor{},
		0,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				if newSensor, ok := new.(*v1alpha1.Sensor); ok {
					sensorCtx.notificationQueue <- &notification{
						sensor:           newSensor,
						notificationType: v1alpha1.ResourceUpdateNotification,
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller
}

func (sensorCtx *sensorContext) newSensorWatch() *cache.ListWatch {
	x := sensorCtx.sensorClient.ArgoprojV1alpha1().RESTClient()
	resource := "sensors"
	name := sensorCtx.sensor.Name
	namespace := sensorCtx.sensor.Namespace
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}
