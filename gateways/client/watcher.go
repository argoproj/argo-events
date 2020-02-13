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

package main

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
func (gatewayContext *GatewayContext) WatchGatewayEventSources(ctx context.Context) (cache.Controller, error) {
	source := gatewayContext.newEventSourceWatch(gatewayContext.eventSourceRef)
	_, controller := cache.NewInformer(
		source,
		&eventSourceV1Alpha1.EventSource{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if newEventSource, ok := obj.(*eventSourceV1Alpha1.EventSource); ok {
					gatewayContext.logger.WithField(common.LabelEventSource, newEventSource.Name).Infoln("detected a new event-source...")
					err := gatewayContext.syncEventSources(newEventSource)
					if err != nil {
						gatewayContext.logger.WithField(common.LabelEventSource, newEventSource.Name).WithError(err).Errorln("failed to process the event-source reference")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if eventSource, ok := new.(*eventSourceV1Alpha1.EventSource); ok {
					gatewayContext.logger.WithField(common.LabelEventSource, eventSource.Name).Info("detected event-source update...")
					err := gatewayContext.syncEventSources(eventSource)
					if err != nil {
						gatewayContext.logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Error("failed to process event source update")
					}
				}
			},
		})
	go controller.Run(ctx.Done())
	return controller, nil
}

// newEventSourceWatch creates a new event source watcher
func (gatewayContext *GatewayContext) newEventSourceWatch(eventSourceRef *v1alpha1.EventSourceRef) *cache.ListWatch {
	client := gatewayContext.eventSourceClient.ArgoprojV1alpha1().RESTClient()
	resource := "eventsources"

	if eventSourceRef.Namespace == "" {
		eventSourceRef.Namespace = gatewayContext.namespace
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
func (gatewayContext *GatewayContext) WatchGatewayUpdates(ctx context.Context) (cache.Controller, error) {
	source := gatewayContext.newGatewayWatch(gatewayContext.name)
	_, controller := cache.NewInformer(
		source,
		&v1alpha1.Gateway{},
		0,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				if g, ok := new.(*v1alpha1.Gateway); ok {
					gatewayContext.logger.Info("detected gateway update. updating gateway watchers")
					gatewayContext.statusCh <- notification{
						gatewayNotification: &resourceUpdate{gateway: g},
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newGatewayWatch creates a new gateway watcher
func (gatewayContext *GatewayContext) newGatewayWatch(name string) *cache.ListWatch {
	x := gatewayContext.gatewayClient.ArgoprojV1alpha1().RESTClient()
	resource := "gateways"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gatewayContext.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gatewayContext.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}
