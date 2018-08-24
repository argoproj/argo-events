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
	"strings"
	"sync"
	"time"

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

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
)

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
// Listen() methods CAN retrieve the kubeConfig from the resource struct.
type resource struct {
	kubeConfig *rest.Config
}

// New creates a new resource signaler
func New(kubeConfig *rest.Config) sdk.Listener {
	return &resource{kubeConfig: kubeConfig}
}

func (r *resource) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	resources, err := r.discoverResources(signal.Resource)
	if err != nil {
		return nil, err
	}

	options := metav1.ListOptions{Watch: true}
	if signal.Resource.Filter != nil {
		options.LabelSelector = labels.Set(signal.Resource.Filter.Labels).AsSelector().String()
	}

	wg := sync.WaitGroup{}
	watches := make([]watch.Interface, 0)
	events := make(chan *v1alpha1.Event)

	// start up listeners
	for i := 0; i < len(resources); i++ {
		resource := resources[i]
		watch, err := resource.Watch(options)
		if err != nil {
			return nil, err
		}
		watches = append(watches, watch)

		wg.Add(1)
		go r.listen(events, watch, signal.Resource.Filter, &wg)
	}

	// wait for stop signal
	go func() {
		<-done
		for _, watch := range watches {
			watch.Stop()
		}
	}()

	// wait for all watches to complete, then close the events channel
	go func() {
		wg.Wait()
		close(events)
	}()

	return events, nil
}

func (r *resource) discoverResources(obj *v1alpha1.ResourceSignal) ([]dynamic.ResourceInterface, error) {
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

func resolveGroupVersion(obj *v1alpha1.ResourceSignal) string {
	if obj.Version == "v1" {
		return obj.Version
	}
	return obj.Group + "/" + obj.Version
}

func (r *resource) listen(events chan *v1alpha1.Event, w watch.Interface, filter *v1alpha1.ResourceFilter, wg *sync.WaitGroup) {
	for item := range w.ResultChan() {
		itemObj := item.Object.(*unstructured.Unstructured)
		b, _ := itemObj.MarshalJSON()
		event := &v1alpha1.Event{
			Context: v1alpha1.EventContext{
				EventTime: metav1.Time{Time: time.Now().UTC()},
			},
			Data: b,
		}

		if item.Type == watch.Error {
			err := errors.FromObject(item.Object)
			log.Panic(err)
		}

		if passFilters(itemObj, filter) {
			events <- event
		}
	}
	wg.Done()
}

// helper method to return a flag indicating if the object passed the client side filters
func passFilters(obj *unstructured.Unstructured, filter *v1alpha1.ResourceFilter) bool {
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
