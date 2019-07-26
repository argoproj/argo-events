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
	"strings"

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
					if err := common.CheckEventSourceVersion(newCm); err != nil {
						gc.Log.WithField("name", newCm.Name).Error(err)
					} else {
						gc.Log.WithField("name", newCm.Name).Info("detected configmap addition")
						err := gc.manageEventSources(newCm)
						if err != nil {
							gc.Log.WithError(err).Error("add config failed")
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if cm, ok := new.(*corev1.ConfigMap); ok {
					if err := common.CheckEventSourceVersion(cm); err != nil {
						gc.Log.WithField("name", cm.Name).Error(err)
					} else {
						gc.Log.Info("detected EventSource update. Updating the controller run config.")
						err := gc.manageEventSources(cm)
						if err != nil {
							gc.Log.WithError(err).Error("update config failed")
						}
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
	namespace := gc.Namespace
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		namespace = parts[0]
		name = parts[1]
	}
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

// WatchGateway watches for changes in the gateway resource
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
					gc.Log.Info("detected gateway update. updating gateway watchers")
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
