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
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"strings"

	"github.com/argoproj/argo-events/gateways"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// StartEventSource starts an event source
func (ese *ResourceEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str(common.LabelEventSource, eventSource.Name).Msg("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		ese.Log.Error().Err(err).Str(common.LabelEventSource, eventSource.Name).Msg("failed to parse event source")
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(config.(*resource), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, ese.Log)
}

// listenEvents watches resource updates and consume those events
func (ese *ResourceEventSourceExecutor) listenEvents(res *resource, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	options := metav1.ListOptions{Watch: true}
	if res.Filter != nil {
		options.LabelSelector = labels.Set(res.Filter.Labels).AsSelector().String()
	}

	ese.Log.WithEventSource(eventSource.Name).Info().Msg("starting to watch to resource notifications")

	resourceList, err := ese.discoverResources(res)
	if err != nil {
		errorCh <- err
		return
	}

	apiResource, err := ese.serverResourceForGVK(resourceList, res.Kind)
	if err != nil {
		errorCh <- err
		return
	}

	if !ese.canWatchResource(apiResource) {
		errorCh <- fmt.Errorf("watch functionality is not allowed on resource")
		return
	}

	dynClientPool := dynamic.NewDynamicClientPool(ese.K8RestConfig)

	gvk := schema.FromAPIVersionAndKind(resourceList.GroupVersion, apiResource.Kind)
	client, err := dynClientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		errorCh <- err
		return
	}

	watcher, err := client.Resource(apiResource, res.Namespace).Watch(options)
	if err != nil {
		errorCh <- err
		return
	}

	watchCh := make(chan struct{})
	// key is group version kind name
	resourceObjects := make(map[string]string)

	localDoneCh := doneCh

	go ese.watchObjectChannel(watcher, res, eventSource, resourceObjects, dataCh, errorCh, watchCh, localDoneCh)

	// renews watch
	// Todo: we shouldn't keep on renewing watch by ourselves but rather use dynamicinformer NewFilteredDynamicSharedInformerFactory https://github.com/kubernetes/client-go/blob/master/dynamic/dynamicinformer/informer.go
	// But this is available from go client release 1.10. It is not possible to upgrade without upgrading Argo, because it has dependency on release 1.9
	// Resolution- Create PR for Argo to upgrade go client version.

	go func() {
		for {
			select {
			case <-watchCh:
				watcher, err := client.Resource(apiResource, res.Namespace).Watch(options)
				if err != nil {
					errorCh <- err
					return
				}
				go ese.watchObjectChannel(watcher, res, eventSource, resourceObjects, dataCh, errorCh, watchCh, localDoneCh)
			case <-localDoneCh:
				return
			}
		}
	}()

	<-doneCh
	close(doneCh)
}

func (ese *ResourceEventSourceExecutor) watchObjectChannel(watcher watch.Interface, res *resource, eventSource *gateways.EventSource, resourceObjects map[string]string, dataCh chan []byte, errorCh chan error, watchCh chan struct{}, doneCh chan struct{}) {
	log := ese.Log.WithEventSource(eventSource.Name)
	for {
		select {
		case item := <-watcher.ResultChan():
			if item.Object == nil {
				log.Info().Msg("watch ended, creating a new watch")
				watchCh <- struct{}{}
				return
			}

			if res.Type != "" && item.Type != res.Type {
				log.Warn().Str("actual-event-type", string(item.Type)).Str("expected-event-type", string(res.Type)).Msg("event type mismatched. won't consume the event")
				continue
			}

			itemObj, isUnst := item.Object.(*unstructured.Unstructured)
			if !isUnst {
				continue
			}
			b, err := itemObj.MarshalJSON()
			if err != nil {
				errorCh <- err
				return
			}
			if item.Type == watch.Error {
				err = errors.FromObject(item.Object)
				errorCh <- err
				return
			}

			resourceKey := fmt.Sprintf("%s%s%s%s", res.GroupVersionKind.Group, res.GroupVersionKind.Version, res.GroupVersionKind.Kind, itemObj.GetName())

			watchedObj, _ := json.Marshal((*itemObj).DeepCopyObject())

			if obj, ok := resourceObjects[resourceKey]; ok {
				if string(watchedObj) == obj {
					log.Info().Msg("update is already watched")
					continue
				}
			}
			resourceObjects[resourceKey] = string(watchedObj)

			if ese.passFilters(eventSource.Name, itemObj, res.Filter) {
				dataCh <- b
			}

		case <-doneCh:
			return
		}
	}
}

func (ese *ResourceEventSourceExecutor) discoverResources(obj *resource) (*metav1.APIResourceList, error) {
	disco, err := discovery.NewDiscoveryClientForConfig(ese.K8RestConfig)
	if err != nil {
		return nil, err
	}
	groupVersion := ese.resolveGroupVersion(obj)
	return disco.ServerResourcesForGroupVersion(groupVersion)
}

func (ese *ResourceEventSourceExecutor) serverResourceForGVK(resourceInterfaces *metav1.APIResourceList, kind string) (*metav1.APIResource, error) {
	for i := range resourceInterfaces.APIResources {
		apiResource := resourceInterfaces.APIResources[i]
		gvk := schema.FromAPIVersionAndKind(resourceInterfaces.GroupVersion, apiResource.Kind)
		ese.Log.Info().Str("api-resource", gvk.String())
		if apiResource.Kind == kind {
			return &apiResource, nil
		}
	}
	ese.Log.Warn().Str("kind", kind).Msg("no resource found")
	return nil, fmt.Errorf("no resource found")
}

func (ese *ResourceEventSourceExecutor) canWatchResource(apiResource *metav1.APIResource) bool {
	for _, verb := range apiResource.Verbs {
		if verb == "watch" {
			return true
		}
	}
	return false
}

func (ese *ResourceEventSourceExecutor) resolveGroupVersion(obj *resource) string {
	if obj.Version == "v1" {
		return obj.Version
	}
	return obj.Group + "/" + obj.Version
}

// helper method to return a flag indicating if the object passed the client side filters
func (ese *ResourceEventSourceExecutor) passFilters(esName string, obj *unstructured.Unstructured, filter *ResourceFilter) bool {
	log := ese.Log.WithEventSource(esName)
	// no filters are applied.
	if filter == nil {
		return true
	}
	// check prefix
	if !strings.HasPrefix(obj.GetName(), filter.Prefix) {
		log.Info().Str("resource-name", obj.GetName()).Str("prefix", filter.Prefix).Msg("FILTERED: resource name does not match prefix")
		return false
	}
	// check creation timestamp
	created := obj.GetCreationTimestamp()
	if !filter.CreatedBy.IsZero() && created.UTC().After(filter.CreatedBy.UTC()) {
		log.Info().Str("creation-timestamp", created.UTC().String()).Str("createdBy", filter.CreatedBy.UTC().String()).Msg("FILTERED: resource creation timestamp is after createdBy")
		return false
	}
	// check labels
	if ok := checkMap(filter.Labels, obj.GetLabels()); !ok {
		log.Info().Interface("resource-labels", obj.GetLabels()).Interface("filter-labels", filter.Labels).Msg("FILTERED: labels mismatch")
		return false
	}
	// check annotations
	if ok := checkMap(filter.Annotations, obj.GetAnnotations()); !ok {
		log.Info().Interface("resource-annotations", obj.GetAnnotations()).Interface("filter-annotations", filter.Annotations).Msg("FILTERED: annotations mismatch")
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
