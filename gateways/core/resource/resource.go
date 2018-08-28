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
	"strings"
	"sync"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"github.com/argoproj/argo-events/gateways"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"fmt"
	"bytes"
	"os"
	zlog "github.com/rs/zerolog"
	hs "github.com/mitchellh/hashstructure"
	apiv1 "k8s.io/api/core/v1"
	"github.com/argoproj/argo-events/common"
	"context"
	"github.com/ghodss/yaml"
)

const (
	configName = "resource-gateway-configmap"
)

type resource struct {
	kubeConfig *rest.Config
	gatewayConfig *gateways.GatewayConfig
	registeredResources map[uint64]*Resource
}

// Resource refers to a dependency on a k8s resource.
type Resource struct {
	Namespace        string          `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	Filter           *ResourceFilter `json:"filter,omitempty" protobuf:"bytes,2,opt,name=filter"`
	metav1.GroupVersionKind `json:",inline" protobuf:"bytes,3,opt,name=groupVersionKind"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource signal objects
type ResourceFilter struct {
	Prefix      string            `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	Labels      map[string]string `json:"labels,omitempty" protobuf:"bytes,2,rep,name=labels"`
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,3,rep,name=annotations"`
	CreatedBy   metav1.Time           `json:"createdBy,omitempty" protobuf:"bytes,4,opt,name=createdBy"`
}

// listens for resource of interest.
func (r *resource) listen(res *Resource) {
	resources, err := r.discoverResources(res)
	if err != nil {
		r.gatewayConfig.Log.Error().Err(err).Msg("failed to discover resource")
		return
	}

	options := metav1.ListOptions{Watch: true}
	if res.Filter != nil {
		options.LabelSelector = labels.Set(res.Filter.Labels).AsSelector().String()
	}

	wg := sync.WaitGroup{}

	// start up listeners
	for i := 0; i < len(resources); i++ {
		resource := resources[i]
		watch, err := resource.Watch(options)
		if err != nil {
			r.gatewayConfig.Log.Error().Err(err).Msg("failed to watch the resource")
			return
		}
		go r.watch(watch, res.Filter, &wg)
	}
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
		log.Printf("found API Resource %s", gvk.String())
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

func (r *resource) watch(w watch.Interface, filter *ResourceFilter, wg *sync.WaitGroup) {
	for item := range w.ResultChan() {
		itemObj := item.Object.(*unstructured.Unstructured)
		b, _ := itemObj.MarshalJSON()
		r.gatewayConfig.Log.Info().Msg("received a request, forwarding it to gateway transformer")
		http.Post(fmt.Sprintf("http://localhost:%s", r.gatewayConfig.TransformerPort), "application/octet-stream", bytes.NewReader(b))

		if item.Type == watch.Error {
			err := errors.FromObject(item.Object)
			log.Panic(err)
		}

		if passFilters(itemObj, filter) {

		}
	}
	wg.Done()
}

// helper method to return a flag indicating if the object passed the client side filters
func passFilters(obj *unstructured.Unstructured, filter *ResourceFilter) bool {
	// check prefix
	if !strings.HasPrefix(obj.GetName(), filter.Prefix) {
		log.Printf("FILTERED: resource name '%s' does not match prefix '%s'", obj.GetName(), filter.Prefix)
		return false
	}
	// check creation timestamp
	created := obj.GetCreationTimestamp()
	if !filter.CreatedBy.IsZero() && created.UTC().After(filter.CreatedBy.UTC()) {
		log.Printf("FILTERED: resource creation timestamp '%s' is after createdBy '%s'", created.UTC(), filter.CreatedBy.UTC())
		return false
	}
	// check labels
	if ok := checkMap(filter.Labels, obj.GetLabels()); !ok {
		log.Printf("FILTERED: resource labels '%s' do not match filter labels '%s'", obj.GetLabels(), filter.Labels)
		return false
	}
	// check annotations
	if ok := checkMap(filter.Annotations, obj.GetAnnotations()); !ok {
		log.Printf("FILTERED: resource annotations '%s' do not match filter annotations '%s'", obj.GetAnnotations(), filter.Annotations)
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

// RunGateway parses and runs each gateway configuration into separate go routines
func (r *resource) RunGateway(cm *apiv1.ConfigMap) error {
	for resourceConfigKey, resourceConfigData := range cm.Data {
		var res *Resource
		err := yaml.Unmarshal([]byte(resourceConfigData), &res)
		if err != nil {
			r.gatewayConfig.Log.Warn().Str("resource-config", resourceConfigKey).Err(err).Msg("failed to parse resource configuration")
			return err
		}
		r.gatewayConfig.Log.Info().Interface("resource", *res)

		key, err := hs.Hash(res, &hs.HashOptions{})

		// check if the gateway is already running this configuration
		if _, ok := r.registeredResources[key]; ok {
			r.gatewayConfig.Log.Warn().Interface("config", res).Msg("duplicate configuration")
			continue
		}
		go r.listen(res)
	}
	return nil
}

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}
	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)

	gatewayConfig := &gateways.GatewayConfig{
		Log:             zlog.New(os.Stdout).With().Logger(),
		Namespace:       namespace,
		Clientset:       clientset,
		TransformerPort: transformerPort,
	}

	res := &resource{
		gatewayConfig:       gatewayConfig,
		registeredResources: make(map[uint64]*Resource),
	}

	// watch the gateway configuration updates
	_, err = gatewayConfig.WatchGatewayConfigMap(res, context.Background(), configName)
	if err != nil {
		res.gatewayConfig.Log.Error().Err(err).Msg("failed to update nats gateway confimap")
	}
	select {}
}