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

	esv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// WatchEventSources watches change in event sources for the gateway
func (gc *GatewayConfig) WatchEventSources(ctx context.Context) (cache.Controller, error) {
	source := gc.newEventSourceWatch(gc.eventSourceResourceName)
	_, controller := cache.NewInformer(
		source,
		&corev1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if es, ok := obj.(*esv1alpha1.EventSource); ok {
					gc.Log.Info().Str("event-source-resource", gc.eventSourceResourceName).Msg("detected event source addition")
					err := gc.manageEventSources(es)
					if err != nil {
						gc.Log.Error().Err(err).Msg("add event source failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if es, ok := new.(*esv1alpha1.EventSource); ok {
					gc.Log.Info().Msg("detected event source update")
					err := gc.manageEventSources(es)
					if err != nil {
						gc.Log.Error().Err(err).Msg("update event source failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newEventSourceWatch creates a new event source watcher
func (gc *GatewayConfig) newEventSourceWatch(name string) *cache.ListWatch {
	x := gc.escs.ArgoprojV1alpha1().RESTClient()
	resource := "eventsources"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// WatchGateway watches for changes in the gateway resource
func (gc *GatewayConfig) WatchGateway(ctx context.Context) (cache.Controller, error) {
	source := gc.newGatewayWatch(gc.Name)
	_, controller := cache.NewInformer(
		source,
		&v1alpha1.Gateway{},
		0,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				if g, ok := new.(*v1alpha1.Gateway); ok {
					gc.Log.Info().Msg("detected gateway update")
					gc.StatusCh <- EventSourceStatus{
						Phase:   v1alpha1.NodePhaseResourceUpdate,
						Gw:      g,
						Message: "gateway_resource_update",
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newGatewayWatch creates a new gateway watcher
func (gc *GatewayConfig) newGatewayWatch(name string) *cache.ListWatch {
	x := gc.gwcs.ArgoprojV1alpha1().RESTClient()
	resource := "gateways"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}
