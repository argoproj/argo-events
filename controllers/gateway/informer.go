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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gatewayinformers "github.com/argoproj/argo-events/pkg/client/gateway/informers/externalversions"
	"k8s.io/apimachinery/pkg/selection"
)

func (c *GatewayController) instanceIDReq() labels.Requirement {
	var instanceIDReq *labels.Requirement
	var err error
	// it makes sense to make instance id is mandatory.
	if c.Config.InstanceID == "" {
		panic("instance id is required")
	}
	instanceIDReq, err = labels.NewRequirement(common.LabelKeyGatewayControllerInstanceID, selection.Equals, []string{c.Config.InstanceID})
	if err != nil {
		panic(err)
	}
	return *instanceIDReq
}

// The gateway-controller informer adds new Gateways to the gateway-controller-controller's queue based on Add, Update, and Delete Event Handlers for the Gateway Resources
func (c *GatewayController) newGatewayInformer() cache.SharedIndexInformer {
	gatewayInformerFactory := gatewayinformers.NewFilteredSharedInformerFactory(
		c.gatewayClientset,
		gatewayResyncPeriod,
		c.Config.Namespace,
		func(options *metav1.ListOptions) {
			options.FieldSelector = fields.Everything().String()
			labelSelector := labels.NewSelector().Add(c.instanceIDReq())
			options.LabelSelector = labelSelector.String()
		},
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
	return informer
}

// newGatewayPodInformer returns a pod informer to watch pods of gateways
func (c *GatewayController) newGatewayPodInformer(informerFactory informers.SharedInformerFactory) informersv1.PodInformer {
	podInformer := informerFactory.Core().V1().Pods()
	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				gw, err := c.getOwnerGateway(obj)
				if err != nil {
					return
				}
				key, err := cache.MetaNamespaceKeyFunc(gw)
				if err == nil {
					// queue the gateway key to examine
					c.queue.Add(key)
				}
			},
		},
	)
	return podInformer
}

// newGatewayServiceInformer returns a service informer to watch services of gateways
func (c *GatewayController) newGatewayServiceInformer(informerFactory informers.SharedInformerFactory) informersv1.ServiceInformer {
	svcInformer := informerFactory.Core().V1().Services()
	svcInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				gw, err := c.getOwnerGateway(obj)
				if err != nil {
					return
				}
				key, err := cache.MetaNamespaceKeyFunc(gw)
				if err == nil {
					// queue the gateway key to examine
					c.queue.Add(key)
				}
			},
		},
	)
	return svcInformer
}

// getOwnerGateway returns gateway of a given resource
func (c *GatewayController) getOwnerGateway(obj interface{}) (*v1alpha1.Gateway, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	for _, owner := range m.GetOwnerReferences() {
		if owner.Kind == gateway.Kind {
			key := owner.Name
			if c.Namespace != "" {
				key = c.Namespace + "/" + key
			}
			obj, exists, err := c.informer.GetIndexer().GetByKey(key)
			if err != nil {
				return nil, err
			}
			if !exists {
				gatewayClient := c.gatewayClientset.ArgoprojV1alpha1().Gateways(c.Namespace)
				gw, err := gatewayClient.Get(owner.Name, metav1.GetOptions{})
				return gw, err
			}
			gw, ok := obj.(*v1alpha1.Gateway)
			if !ok {
				return nil, fmt.Errorf("failed to cast gateway")
			}
			return gw, nil
		}
	}
	return nil, fmt.Errorf("no gateway owner found")
}
