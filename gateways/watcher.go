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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// WatchGatewayEventSources watches change in configuration for the gateway
func (gc *GatewayConfig) WatchGatewayEventSources(ctx context.Context) (cache.Controller, error) {
	source := gc.newConfigMapWatch(gc.configName)
	_, controller := cache.NewInformer(
		source,
		&corev1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if newCm, ok := obj.(*corev1.ConfigMap); ok {
					gc.Log.Info().Str("config-map", gc.configName).Msg("detected ConfigMap addition. Updating the controller run config.")
					err := gc.manageEventSources(newCm)
					if err != nil {
						gc.Log.Error().Err(err).Msg("add config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if cm, ok := new.(*corev1.ConfigMap); ok {
					gc.Log.Info().Msg("detected ConfigMap update. Updating the controller run config.")
					err := gc.manageEventSources(cm)
					if err != nil {
						gc.Log.Error().Err(err).Msg("update config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newConfigMapWatch creates a new configmap watcher
func (gc *GatewayConfig) newConfigMapWatch(name string) *cache.ListWatch {
	x := gc.Clientset.CoreV1().RESTClient()
	resource := "configmaps"
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

// WatchGatewayEventSources watches change in the gateway resource
// This will act as replacement for old gateway-transformer-configmap. Changes to watchers, event version and event type will be reflected.
func (gc *GatewayConfig) WatchGateway(ctx context.Context) (cache.Controller, error) {
	source := gc.newGatewayWatch(gc.Name)
	_, controller := cache.NewInformer(
		source,
		&v1alpha1.Gateway{},
		0,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				if g, ok := new.(*v1alpha1.Gateway); ok {
					gc.Log.Info().Msg("detected gateway update. updating gateway watchers and event metadata")
					gc.StatusCh <- EventSourceStatus{
						Phase: v1alpha1.NodePhaseResourceUpdate,
						Gw: g,
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
