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

package resource

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"strings"
)

// StartConfig runs a configuration
func (ce *ResourceConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	defer func() {
		gateways.Recover()
	}()

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	res, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *res).Msg("resource configuration")

	go ce.listenEvents(res, config)

	for {
		select {
		case _, ok :=<-config.StartChan:
			if ok {
				ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running.")
				config.Active = true
			}

		case data, ok := <-config.DataChan:
			if ok {
				err := ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: data,
				})
				if err != nil {
					config.ErrChan <- err
				}
			}

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *ResourceConfigExecutor) listenEvents(res *resource, config *gateways.ConfigContext) {
	resources, err := ce.discoverResources(res)
	if err != nil {
		config.ErrChan <- err
		return
	}
	options := metav1.ListOptions{Watch: true}
	if res.Filter != nil {
		options.LabelSelector = labels.Set(res.Filter.Labels).AsSelector().String()
	}

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("k8 event created marking configuration as running")

	config.StartChan <- struct{}{}

	// global quit channel
	quitChan := make(chan struct{})

	// start up listeners
	for i := 0; i < len(resources); i++ {
		resource := resources[i]
		w, err := resource.Watch(options)
		if err != nil {
			config.ErrChan <- err
			return
		}

		localQuitChan := quitChan

		go func() {
			for {
				select {
				case item := <-w.ResultChan():
					if item.Object == nil {
						ce.Log.Warn().Str("config-key", config.Data.Src).Msg("object to watch is nil")
						return
					}
					itemObj := item.Object.(*unstructured.Unstructured)
					b, err := itemObj.MarshalJSON()
					if err != nil {
						config.ErrChan <- err
					}
					if item.Type == watch.Error {
						err = errors.FromObject(item.Object)
						config.ErrChan <- err
					}
					if ce.passFilters(itemObj, res.Filter) {
						config.DataChan <- b
					}

				case <-localQuitChan:
					return
				}
			}
		}()
	}

	<-config.DoneChan
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
	close(quitChan)
	config.ShutdownChan <- struct{}{}
	return
}

func (ce *ResourceConfigExecutor) discoverResources(obj *resource) ([]dynamic.ResourceInterface, error) {
	dynClientPool := dynamic.NewDynamicClientPool(ce.GatewayConfig.KubeConfig)
	disco, err := discovery.NewDiscoveryClientForConfig(ce.GatewayConfig.KubeConfig)
	if err != nil {
		return nil, err
	}

	groupVersion := ce.resolveGroupVersion(obj)
	resourceInterfaces, err := disco.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return nil, err
	}
	resources := make([]dynamic.ResourceInterface, 0)
	for i := range resourceInterfaces.APIResources {
		apiResource := resourceInterfaces.APIResources[i]
		gvk := schema.FromAPIVersionAndKind(resourceInterfaces.GroupVersion, apiResource.Kind)
		ce.GatewayConfig.Log.Info().Str("api-resource", gvk.String())
		if apiResource.Kind != obj.Kind {
			continue
		}
		canWatch := false
		for _, verb := range apiResource.Verbs {
			if verb == "watch" {
				canWatch = true
				break
			}
		}
		if canWatch {
			client, err := dynClientPool.ClientForGroupVersionKind(gvk)
			if err != nil {
				return nil, err
			}
			resources = append(resources, client.Resource(&apiResource, obj.Namespace))
		}
	}
	return resources, nil
}

func (ce *ResourceConfigExecutor) resolveGroupVersion(obj *resource) string {
	if obj.Version == "v1" {
		return obj.Version
	}
	return obj.Group + "/" + obj.Version
}

// helper method to return a flag indicating if the object passed the client side filters
func (ce *ResourceConfigExecutor) passFilters(obj *unstructured.Unstructured, filter *ResourceFilter) bool {
	// no filters are applied.
	if filter == nil {
		return true
	}
	// check prefix
	if !strings.HasPrefix(obj.GetName(), filter.Prefix) {
		ce.GatewayConfig.Log.Info().Str("resource-name", obj.GetName()).Str("prefix", filter.Prefix).Msg("FILTERED: resource name does not match prefix")
		return false
	}
	// check creation timestamp
	created := obj.GetCreationTimestamp()
	if !filter.CreatedBy.IsZero() && created.UTC().After(filter.CreatedBy.UTC()) {
		ce.GatewayConfig.Log.Info().Str("creation-timestamp", created.UTC().String()).Str("createdBy", filter.CreatedBy.UTC().String()).Msg("FILTERED: resource creation timestamp is after createdBy")
		return false
	}
	// check labels
	if ok := checkMap(filter.Labels, obj.GetLabels()); !ok {
		ce.GatewayConfig.Log.Info().Interface("resource-labels", obj.GetLabels()).Interface("filter-labels", filter.Labels).Msg("FILTERED: labels mismatch")
		return false
	}
	// check annotations
	if ok := checkMap(filter.Annotations, obj.GetAnnotations()); !ok {
		ce.GatewayConfig.Log.Info().Interface("resource-annotations", obj.GetAnnotations()).Interface("filter-annotations", filter.Annotations).Msg("FILTERED: annotations mismatch")
		return false
	}
	return true
}

// utility method to check the actual map matches the expected by values
func checkMap(expected, actual map[string]string) bool {
	if actual != nil {
		for k, v := range expected {
			if actual[k] != v {
				return false
			}
		}
		return true
	}
	if expected != nil {
		return false
	}
	return true
}
