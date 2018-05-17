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
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/blackrock/axis/job"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
)

type resource struct {
	job.AbstractSignal
	kubeConfig *rest.Config
	watches    []watch.Interface
}

func (r *resource) Start(events chan job.Event) error {
	var err error
	dynClientPool := dynamic.NewDynamicClientPool(r.kubeConfig)
	disco, err := discovery.NewDiscoveryClientForConfig(r.kubeConfig)
	if err != nil {
		return err
	}
	var groupVersion string
	if r.Resource.Version == "v1" {
		groupVersion = "v1"
	} else {
		groupVersion = r.Resource.Group + "/" + r.Resource.Version
	}
	resourceInterfaces, err := disco.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return err
	}

	resources := make([]dynamic.ResourceInterface, 0)
	for i := range resourceInterfaces.APIResources {
		apiResource := resourceInterfaces.APIResources[i]
		gvk := schema.FromAPIVersionAndKind(resourceInterfaces.GroupVersion, apiResource.Kind)
		r.Log.Info("found API Resource", zap.String("API Group Version", gvk.String()))
		if apiResource.Kind != r.Resource.Kind {
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
				return err
			}
			resources = append(resources, client.Resource(&apiResource, r.Resource.Namespace))
		}
	}

	options := metav1.ListOptions{Watch: true}
	if r.Resource.Filter != nil {
		options.LabelSelector = labels.Set(r.Resource.Filter.Labels).AsSelector().String()
	}

	for i := 0; i < len(resources); i++ {
		resource := resources[i]
		watch, err := resource.Watch(options)
		if err != nil {
			return err
		}
		r.watches = append(r.watches, watch)
		go r.listen(events, watch)
	}

	return nil
}

func (r *resource) Stop() error {
	for _, watch := range r.watches {
		watch.Stop()
	}
	return nil
}

func (r *resource) listen(events chan job.Event, w watch.Interface) {
	for item := range w.ResultChan() {
		itemObj := item.Object.(*unstructured.Unstructured)
		event := &event{
			obj:       itemObj,
			timestamp: time.Now().UTC(),
			resource:  r,
		}

		r.Log.Info("processing watch event", zap.ByteString("body", event.GetBody()))

		if item.Type == watch.Error {
			err := errors.FromObject(item.Object)
			r.Log.Warn("watch error encountered", zap.Error(err))
			event.SetError(err)
		}

		err := r.CheckConstraints(event.GetTimestamp())
		if err != nil {
			event.SetError(err)
		}

		if r.passFilters(itemObj, r.Resource.Filter) {
			events <- event
		}
	}
}

// helper method to return a flag indicating if the object passed the client side filters
func (r *resource) passFilters(obj *unstructured.Unstructured, filter *v1alpha1.ResourceFilter) bool {
	// check prefix
	if !strings.HasPrefix(obj.GetName(), filter.Prefix) {
		r.Log.Debug("FILTERED: resource name does not match prefix", zap.String("name", obj.GetName()), zap.String("prefix", filter.Prefix))
		return false
	}
	// check creation timestamp
	created := obj.GetCreationTimestamp()
	if !filter.CreatedBy.IsZero() && created.UTC().After(filter.CreatedBy.UTC()) {
		r.Log.Debug("FILTERED: resource creation timestamp is after createdBy", zap.Time("obj creation", created.UTC()), zap.Time("createdBy", filter.CreatedBy.UTC()))
		return false
	}
	// check labels
	if ok := checkMap(filter.Labels, obj.GetLabels()); !ok {
		r.Log.Debug("FILTERED: resource labels do not match filter labels")
		return false
	}
	// check annotations
	if ok := checkMap(filter.Annotations, obj.GetAnnotations()); !ok {
		r.Log.Debug("FILTERED: resource annotations do not match filter annotations")
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
