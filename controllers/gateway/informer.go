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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/cache"

	gatewayinformers "github.com/argoproj/argo-events/pkg/client/gateway/informers/externalversions"
)

func (c *Controller) instanceIDReq() (*labels.Requirement, error) {
	if c.Config.InstanceID == "" {
		panic("instance id is required")
	}
	instanceIDReq, err := labels.NewRequirement(LabelControllerInstanceID, selection.Equals, []string{c.Config.InstanceID})
	if err != nil {
		return nil, err
	}
	c.logger.WithField("instance-id", instanceIDReq.String()).Infoln("instance id requirement")
	return instanceIDReq, nil
}

// The controller informer adds new gateways to the controller's queue based on Add, Update, and Delete Event Handlers for the gateway Resources
func (c *Controller) newGatewayInformer() (cache.SharedIndexInformer, error) {
	labelSelector, err := c.instanceIDReq()
	if err != nil {
		return nil, err
	}
	gatewayInformerFactory := gatewayinformers.NewSharedInformerFactoryWithOptions(
		c.gatewayClient,
		gatewayResyncPeriod,
		gatewayinformers.WithNamespace(c.Config.Namespace),
		gatewayinformers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.Everything().String()
			options.LabelSelector = labelSelector.String()
		}),
	)
	informer := gatewayInformerFactory.Argoproj().V1alpha1().Gateways().Informer()
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
	return informer, nil
}
