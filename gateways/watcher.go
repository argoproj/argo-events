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

package gateways

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	eventSourceV1Alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// WatchGatewayEventSources watches change in event source for the gateway
func (gatewayCfg *GatewayConfig) WatchGatewayEventSources(ctx context.Context) (cache.Controller, error) {
	source := gatewayCfg.newEventSourceWatch(gatewayCfg.eventSourceRef)
	_, controller := cache.NewInformer(
		source,
		&eventSourceV1Alpha1.EventSource{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if newEventSource, ok := obj.(*eventSourceV1Alpha1.EventSource); ok {
					gatewayCfg.Logger.WithField(common.LabelEventSource, newEventSource.Name).Infoln("detected a new event-source...")
					err := gatewayCfg.manageEventSources(newEventSource)
					if err != nil {
						gatewayCfg.Logger.WithField(common.LabelEventSource, newEventSource.Name).WithError(err).Errorln("failed to process the event-source reference")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if eventSource, ok := new.(*eventSourceV1Alpha1.EventSource); ok {
					gatewayCfg.Logger.WithField(common.LabelEventSource, eventSource.Name).Info("detected event-source update...")
					err := gatewayCfg.manageEventSources(eventSource)
					if err != nil {
						gatewayCfg.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Error("failed to process event source update")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newEventSourceWatch creates a new event source watcher
func (gatewayCfg *GatewayConfig) newEventSourceWatch(eventSourceRef *v1alpha1.EventSourceRef) *cache.ListWatch {
	client := gatewayCfg.EventSourceClient.ArgoprojV1alpha1().RESTClient()
	resource := "eventsources"

	if eventSourceRef.Namespace == "" {
		eventSourceRef.Namespace = gatewayCfg.Namespace
	}

	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", eventSourceRef.Name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := client.Get().
			Namespace(eventSourceRef.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := client.Get().
			Namespace(eventSourceRef.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// WatchGatewayUpdates watches for changes in the gateway resource
func (gatewayCfg *GatewayConfig) WatchGatewayUpdates(ctx context.Context) (cache.Controller, error) {
	source := gatewayCfg.newGatewayWatch(gatewayCfg.Name)
	_, controller := cache.NewInformer(
		source,
		&v1alpha1.Gateway{},
		0,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				if g, ok := new.(*v1alpha1.Gateway); ok {
					gatewayCfg.Logger.Info("detected gateway update. updating gateway watchers")
					gatewayCfg.StatusCh <- EventSourceStatus{
						Phase:   v1alpha1.NodePhaseResourceUpdate,
						Gateway: g,
						Message: "gateway_resource_update",
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newGatewayWatch creates a new gateway watcher
func (gatewayCfg *GatewayConfig) newGatewayWatch(name string) *cache.ListWatch {
	x := gatewayCfg.gatewayClient.ArgoprojV1alpha1().RESTClient()
	resource := "gateways"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gatewayCfg.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gatewayCfg.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}
