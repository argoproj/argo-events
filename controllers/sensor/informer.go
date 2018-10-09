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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-events/common"
	sensorinformers "github.com/argoproj/argo-events/pkg/client/sensor/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/selection"
)

func (c *SensorController) instanceIDReq() labels.Requirement {
	var instanceIDReq *labels.Requirement
	var err error
	if c.Config.InstanceID == "" {
		panic("controller instance id must be specified")
	}
	instanceIDReq, err = labels.NewRequirement(common.LabelKeySensorControllerInstanceID, selection.Equals, []string{c.Config.InstanceID})
	if err != nil {
		panic(err)
	}
	return *instanceIDReq
}

// The sensor informer adds new Sensors to the sensor-controller's queue based on Add, Update, and Delete Event Handlers for the Sensor Resources
func (c *SensorController) newSensorInformer() cache.SharedIndexInformer {
	sensorInformerFactory := sensorinformers.NewFilteredSharedInformerFactory(
		c.sensorClientset,
		sensorResyncPeriod,
		// user may need to deploy controller only in one namespace but would like to have sensors created in different namespaces.
		corev1.NamespaceAll,
		func(options *metav1.ListOptions) {
			options.FieldSelector = fields.Everything().String()

			labelSelector := labels.NewSelector().Add(c.instanceIDReq())
			options.LabelSelector = labelSelector.String()
		},
	)
	informer := sensorInformerFactory.Argoproj().V1alpha1().Sensors().Informer()
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					c.queue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					c.queue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					c.queue.Add(key)
				}
			},
		},
	)
	return informer
}
