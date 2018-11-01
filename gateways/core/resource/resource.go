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
	"sync"
	"fmt"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// resourceConfigExecutor implements ConfigExecutor interface
type resourceConfigExecutor struct{}

// StartConfig runs a configuration
func (rce *resourceConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	res := config.Data.Config.(*resource)
	gatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *res).Msg("resource configuration")

	resources, err := discoverResources(res)
	if err != nil {
		errMessage = "failed to discover resource"
		return err
	}
	options := metav1.ListOptions{Watch: true}
	if res.Filter != nil {
		options.LabelSelector = labels.Set(res.Filter.Labels).AsSelector().String()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	// waits till disconnection from client.
	go func() {
		<-config.StopCh
		config.Active = false
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")
		wg.Done()
	}()

	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	// start up listeners
	for i := 0; i < len(resources); i++ {
		resource := resources[i]
		w, err := resource.Watch(options)
		if err != nil {
			errMessage = "failed to watch the resource"
			return err
		}
		go func() {
			for item := range w.ResultChan() {
				itemObj := item.Object.(*unstructured.Unstructured)
				b, err := itemObj.MarshalJSON()
				gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("resource notification")
				if err != nil {
					errMessage = "failed to marshal resource"
					return
				}
				if item.Type == watch.Error {
					err = errors.FromObject(item.Object)
					errMessage = "failed to watch resource"
					return
				}
				if passFilters(itemObj, res.Filter) {
					gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
						Src:     config.Data.Src,
						Payload: b,
					})
				}
			}
		}()
	}
	wg.Wait()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// StopConfig stops a configuration
func (rce *resourceConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

// Validate validates gateway configuration
func (rce *resourceConfigExecutor) Validate(config *gateways.ConfigContext) error {
	res, ok := config.Data.Config.(*resource)
	if !ok {
		return gateways.ErrConfigParseFailed
	}
	if res.Version == "" {
		return fmt.Errorf("%+v, resource version must be specified", gateways.ErrInvalidConfig)
	}
	if res.Namespace == "" {
		return fmt.Errorf("%+v, resource namespace must be specified", gateways.ErrInvalidConfig)
	}
	if res.Kind == "" {
		return fmt.Errorf("%+v, resource kind must be specified", gateways.ErrInvalidConfig)
	}
	if res.Group == "" {
		return fmt.Errorf("%+v, resource group must be specified", gateways.ErrInvalidConfig)
	}
	return nil
}

func discoverResources(obj *resource) ([]dynamic.ResourceInterface, error) {
	dynClientPool := dynamic.NewDynamicClientPool(gatewayConfig.KubeConfig)
	disco, err := discovery.NewDiscoveryClientForConfig(gatewayConfig.KubeConfig)
	if err != nil {
		return nil, err
	}

	groupVersion := resolveGroupVersion(obj)
	resourceInterfaces, err := disco.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return nil, err
	}
	resources := make([]dynamic.ResourceInterface, 0)
	for i := range resourceInterfaces.APIResources {
		apiResource := resourceInterfaces.APIResources[i]
		gvk := schema.FromAPIVersionAndKind(resourceInterfaces.GroupVersion, apiResource.Kind)
		gatewayConfig.Log.Info().Str("api-resource", gvk.String())
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

func resolveGroupVersion(obj *resource) string {
	if obj.Version == "v1" {
		return obj.Version
	}
	return obj.Group + "/" + obj.Version
}

// helper method to return a flag indicating if the object passed the client side filters
func passFilters(obj *unstructured.Unstructured, filter *ResourceFilter) bool {
	// check prefix
	if !strings.HasPrefix(obj.GetName(), filter.Prefix) {
		gatewayConfig.Log.Info().Str("resource-name", obj.GetName()).Str("prefix", filter.Prefix).Msg("FILTERED: resource name does not match prefix")
		return false
	}
	// check creation timestamp
	created := obj.GetCreationTimestamp()
	if !filter.CreatedBy.IsZero() && created.UTC().After(filter.CreatedBy.UTC()) {
		gatewayConfig.Log.Info().Str("creation-timestamp", created.UTC().String()).Str("createdBy", filter.CreatedBy.UTC().String()).Msg("FILTERED: resource creation timestamp is after createdBy")
		return false
	}
	// check labels
	if ok := checkMap(filter.Labels, obj.GetLabels()); !ok {
		gatewayConfig.Log.Info().Interface("resource-labels", obj.GetLabels()).Interface("filter-labels", filter.Labels).Msg("FILTERED: labels mismatch")
		return false
	}
	// check annotations
	if ok := checkMap(filter.Annotations, obj.GetAnnotations()); !ok {
		gatewayConfig.Log.Info().Interface("resource-annotations", obj.GetAnnotations()).Interface("filter-annotations", filter.Annotations).Msg("FILTERED: annotations mismatch")
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

func main() {
	gatewayConfig.StartGateway(&resourceConfigExecutor{})
}
