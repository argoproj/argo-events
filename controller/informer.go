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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-events/common"
	sensorinformers "github.com/argoproj/argo-events/pkg/client/informers/externalversions"
)

func (c *SensorController) instanceIDReq() labels.Requirement {
	var instanceIDReq *labels.Requirement
	var err error
	if c.Config.InstanceID != "" {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeySensorControllerInstanceID, selection.Equals, []string{c.Config.InstanceID})
	} else {
		instanceIDReq, err = labels.NewRequirement(common.LabelKeySensorControllerInstanceID, selection.DoesNotExist, nil)
	}
	if err != nil {
		panic(err)
	}

	return *instanceIDReq
}

// The sensor informer adds new Sensors to the controller's queue based on Add, Update, and Delete Event Handlers for the Sensor Resources
func (c *SensorController) newSensorInformer() cache.SharedIndexInformer {
	sensorInformerFactory := sensorinformers.NewFilteredSharedInformerFactory(
		c.sensorClientset,
		sensorResyncPeriod,
		c.Config.Namespace,
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
					c.ssQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					c.ssQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					c.ssQueue.Add(key)
				}
			},
		},
	)
	return informer
}

func (c *SensorController) newPodInformer() cache.SharedIndexInformer {
	restClient := c.kubeClientset.CoreV1().RESTClient()
	// selectors
	fieldSelector := fields.ParseSelectorOrDie("status.phase!=Pending")
	incompleteReq, _ := labels.NewRequirement(common.LabelKeyResolved, selection.Equals, []string{"false"})
	labelSelector := labels.NewSelector().Add(*incompleteReq).Add(c.instanceIDReq())

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector.String()
				options.LabelSelector = labelSelector.String()
				req := restClient.Get().Namespace(c.Config.Namespace).Resource("pods").VersionedParams(&options, metav1.ParameterCodec)
				return req.Do().Get()
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.Watch = true
				options.FieldSelector = fieldSelector.String()
				options.LabelSelector = labelSelector.String()
				req := restClient.Get().Namespace(c.Config.Namespace).Resource("pods").VersionedParams(&options, metav1.ParameterCodec)
				return req.Watch()
			},
		},
		&corev1.Pod{},
		sensorResyncPeriod,
		cache.Indexers{},
	)
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					c.podQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					c.podQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					c.podQueue.Add(key)
				}
			},
		},
	)
	return informer
}
