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
	gateways "github.com/argoproj/argo-events/gateways/core"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"strings"
	"sync"
)

type resource struct {
	kubeConfig          *rest.Config
	gatewayConfig       *gateways.GatewayConfig
	registeredResources map[uint64]*Resource
}

// Resource refers to a dependency on a k8s resource.
type Resource struct {
	Namespace               string          `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	Filter                  *ResourceFilter `json:"filter,omitempty" protobuf:"bytes,2,opt,name=filter"`
	metav1.GroupVersionKind `json:",inline" protobuf:"bytes,3,opt,name=groupVersionKind"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource signal objects
type ResourceFilter struct {
	Prefix      string            `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	Labels      map[string]string `json:"labels,omitempty" protobuf:"bytes,2,rep,name=labels"`
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,3,rep,name=annotations"`
	CreatedBy   metav1.Time       `json:"createdBy,omitempty" protobuf:"bytes,4,opt,name=createdBy"`
}

// listens for resource of interest.
func (r *resource) RunConfiguration(config *gateways.ConfigData) error {
	r.gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("parsing configuration...")

	var res *Resource
	err := yaml.Unmarshal([]byte(config.Config), &res)
	if err != nil {
		r.gatewayConfig.Log.Warn().Str("config-key", config.Src).Err(err).Msg("failed to parse resource configuration")
		return err
	}

	resources, err := r.discoverResources(res)
	if err != nil {
		r.gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to discover resource")
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
		r.gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("stopping the configuration...")
		wg.Done()
	}()

	config.Active = true
	// start up listeners
	for i := 0; i < len(resources); i++ {
		resource := resources[i]
		w, err := resource.Watch(options)
		if err != nil {
			r.gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to watch the resource")
			return err
		}

		go func() {
			for item := range w.ResultChan() {
				itemObj := item.Object.(*unstructured.Unstructured)
				b, _ := itemObj.MarshalJSON()
				if item.Type == watch.Error {
					err := errors.FromObject(item.Object)
					r.gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to watch resource")
				}
				if r.passFilters(itemObj, res.Filter) {
					r.gatewayConfig.DispatchEvent(b, config.Src)
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

func (r *resource) discoverResources(obj *Resource) ([]dynamic.ResourceInterface, error) {
	dynClientPool := dynamic.NewDynamicClientPool(r.kubeConfig)
	disco, err := discovery.NewDiscoveryClientForConfig(r.kubeConfig)
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
		r.gatewayConfig.Log.Info().Str("api-resource", gvk.String())
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

func resolveGroupVersion(obj *Resource) string {
	if obj.Version == "v1" {
		return obj.Version
	}
	return obj.Group + "/" + obj.Version
}

// helper method to return a flag indicating if the object passed the client side filters
func (r *resource) passFilters(obj *unstructured.Unstructured, filter *ResourceFilter) bool {
	// check prefix
	if !strings.HasPrefix(obj.GetName(), filter.Prefix) {
		r.gatewayConfig.Log.Info().Str("resource-name", obj.GetName()).Str("prefix", filter.Prefix).Msg("FILTERED: resource name does not match prefix")
		return false
	}
	// check creation timestamp
	created := obj.GetCreationTimestamp()
	if !filter.CreatedBy.IsZero() && created.UTC().After(filter.CreatedBy.UTC()) {
		r.gatewayConfig.Log.Info().Str("creation-timestamp", created.UTC().String()).Str("createdBy", filter.CreatedBy.UTC().String()).Msg("FILTERED: resource creation timestamp is after createdBy")
		return false
	}
	// check labels
	if ok := checkMap(filter.Labels, obj.GetLabels()); !ok {
		r.gatewayConfig.Log.Info().Interface("resource-labels", obj.GetLabels()).Interface("filter-labels", filter.Labels).Msg("FILTERED: labels mismatch")
		return false
	}
	// check annotations
	if ok := checkMap(filter.Annotations, obj.GetAnnotations()); !ok {
		r.gatewayConfig.Log.Info().Interface("resource-annotations", obj.GetAnnotations()).Interface("filter-annotations", filter.Annotations).Msg("FILTERED: annotations mismatch")
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
	gatewayConfig := gateways.NewGatewayConfiguration()
	r := &resource{
		gatewayConfig: gatewayConfig,
	}
	gatewayConfig.WatchGatewayConfigMap(r, context.Background())
	select {}
}
