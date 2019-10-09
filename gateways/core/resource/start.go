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
	"strings"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// StartEventSource starts an event source
func (executor *ResourceEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	log := executor.Log.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("operating on event source")

	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go executor.listenEvents(config.(*resource), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, executor.Log)
}

// listenEvents watches resource updates and consume those events
func (executor *ResourceEventSourceExecutor) listenEvents(resourceCfg *resource, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	executor.Log.WithField(common.LabelEventSource, eventSource.Name).Info("started listening resource notifications")

	client, err := dynamic.NewForConfig(executor.K8RestConfig)
	if err != nil {
		errorCh <- err
		return
	}

	gvr := schema.GroupVersionResource{
		Group:    resourceCfg.Group,
		Version:  resourceCfg.Version,
		Resource: resourceCfg.Resource,
	}

	client.Resource(gvr)

	options := &metav1.ListOptions{}

	if resourceCfg.Filter != nil && resourceCfg.Filter.Labels != nil {
		sel, err := LabelSelector(resourceCfg.Filter.Labels)
		if err != nil {
			errorCh <- err
			return
		}
		options.LabelSelector = sel.String()
	}

	if resourceCfg.Filter != nil && resourceCfg.Filter.Fields != nil {
		sel, err := LabelSelector(resourceCfg.Filter.Fields)
		if err != nil {
			errorCh <- err
			return
		}
		options.FieldSelector = sel.String()
	}

	tweakListOptions := func(op *metav1.ListOptions) {
		op = options
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, 0, resourceCfg.Namespace, tweakListOptions)

	informer := factory.ForResource(gvr)

	informerEventCh := make(chan *InformerEvent)

	go func() {
		for {
			select {
			case event, ok := <-informerEventCh:
				if !ok {
					return
				}
				eventBody, err := json.Marshal(event)
				if err != nil {
					executor.Log.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to parse event from resource informer")
					continue
				}
				if err := passFilters(event.Obj.(*unstructured.Unstructured), resourceCfg.Filter); err != nil {
					executor.Log.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Warnln("failed to apply the filter")
					continue
				}
				dataCh <- eventBody
			}
		}
	}()

	sharedInformer := informer.Informer()
	sharedInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				informerEventCh <- &InformerEvent{
					Obj:  obj,
					Type: ADD,
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				informerEventCh <- &InformerEvent{
					Obj:    newObj,
					OldObj: oldObj,
					Type:   UPDATE,
				}
			},
			DeleteFunc: func(obj interface{}) {
				informerEventCh <- &InformerEvent{
					Obj:  obj,
					Type: DELETE,
				}
			},
		},
	)

	sharedInformer.Run(doneCh)
	executor.Log.WithField(common.LabelEventSource, eventSource.Name).Infoln("resource informer is stopped")
	close(informerEventCh)
	close(doneCh)
}

// LabelReq returns label requirements
func LabelReq(key, value string) (*labels.Requirement, error) {
	req, err := labels.NewRequirement(key, selection.Equals, []string{value})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// LabelSelector returns label selector for resource filtering
func LabelSelector(resourceLabels map[string]string) (labels.Selector, error) {
	var labelRequirements []labels.Requirement
	for key, value := range resourceLabels {
		req, err := LabelReq(key, value)
		if err != nil {
			return nil, err
		}
		labelRequirements = append(labelRequirements, *req)
	}
	return labels.NewSelector().Add(labelRequirements...), nil
}

// FieldSelector returns field selector for resource filtering
func FieldSelector(fieldSelectors map[string]string) (fields.Selector, error) {
	var selectors []fields.Selector
	for key, value := range fieldSelectors {
		selector, err := fields.ParseSelector(fmt.Sprintf("%s=%s", key, value))
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, selector)
	}
	return fields.AndSelectors(selectors...), nil
}

// helper method to check if the object passed the user defined filters
func passFilters(obj *unstructured.Unstructured, filter *ResourceFilter) error {
	// no filters are applied.
	if filter == nil {
		return nil
	}
	if !strings.HasPrefix(obj.GetName(), filter.Prefix) {
		return errors.Errorf("resource name does not match prefix. resource-name: %s, prefix: %s", obj.GetName(), filter.Prefix)
	}
	created := obj.GetCreationTimestamp()
	if !filter.CreatedBy.IsZero() && created.UTC().After(filter.CreatedBy.UTC()) {
		return errors.Errorf("resource is created after filter time. creation-timestamp: %s, filter-creation-timestamp: %s", created.UTC().String(), filter.CreatedBy.UTC().String())
	}
	return nil
}
