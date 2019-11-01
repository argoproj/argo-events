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
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// InformerEvent holds event generated from resource state change
type InformerEvent struct {
	Obj    interface{}
	OldObj interface{}
	Type   v1alpha1.ResourceEventType
}

// EventListener implements Eventing
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
	// K8RestConfig is kubernetes cluster config
	K8RestConfig *rest.Config
}

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("activating the event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents watches resource updates and consume those events
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("started processing the event source...")

	logger.Infoln("parsing resource event source...")
	var resourceEventSource *v1alpha1.ResourceEventSource
	if err := yaml.Unmarshal(eventSource.Value, &resourceEventSource); err != nil {
		errorCh <- err
		return
	}

	logger.Infoln("setting up a K8s client")
	client, err := dynamic.NewForConfig(listener.K8RestConfig)
	if err != nil {
		errorCh <- err
		return
	}

	gvr := schema.GroupVersionResource{
		Group:    resourceEventSource.Group,
		Version:  resourceEventSource.Version,
		Resource: resourceEventSource.Resource,
	}

	client.Resource(gvr)

	options := &metav1.ListOptions{}

	logger.Infoln("configuring label selectors if filters are selected...")
	if resourceEventSource.Filter != nil && resourceEventSource.Filter.Labels != nil {
		sel, err := LabelSelector(resourceEventSource.Filter.Labels)
		if err != nil {
			errorCh <- err
			return
		}
		options.LabelSelector = sel.String()
	}

	if resourceEventSource.Filter != nil && resourceEventSource.Filter.Fields != nil {
		sel, err := LabelSelector(resourceEventSource.Filter.Fields)
		if err != nil {
			errorCh <- err
			return
		}
		options.FieldSelector = sel.String()
	}

	tweakListOptions := func(op *metav1.ListOptions) {
		op = options
	}

	logger.Infoln("setting up informer factory...")
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, 0, resourceEventSource.Namespace, tweakListOptions)

	informer := factory.ForResource(gvr)

	informerEventCh := make(chan *InformerEvent)

	go func() {
		logger.Infoln("listening to resource events...")
		for {
			select {
			case event, ok := <-informerEventCh:
				if !ok {
					return
				}
				eventBody, err := json.Marshal(event)
				if err != nil {
					logger.WithError(err).Errorln("failed to parse event from resource informer")
					continue
				}
				if err := passFilters(event.Obj.(*unstructured.Unstructured), resourceEventSource.Filter); err != nil {
					logger.WithError(err).Warnln("failed to apply the filter")
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
					Type: v1alpha1.ADD,
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				informerEventCh <- &InformerEvent{
					Obj:    newObj,
					OldObj: oldObj,
					Type:   v1alpha1.UPDATE,
				}
			},
			DeleteFunc: func(obj interface{}) {
				informerEventCh <- &InformerEvent{
					Obj:  obj,
					Type: v1alpha1.DELETE,
				}
			},
		},
	)

	sharedInformer.Run(doneCh)
	logger.Infoln("resource informer is stopped")
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
func passFilters(obj *unstructured.Unstructured, filter *v1alpha1.ResourceFilter) error {
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
