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
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/events"
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
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")
	channels := server.NewChannels()

	go server.HandleEventsFromEventSource(eventSource.Name, eventStream, channels, listener.Logger)

	defer func() {
		channels.Stop <- struct{}{}
	}()

	if err := listener.listenEvents(eventSource, channels); err != nil {
		listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to listen to events")
		return err
	}

	return nil
}

// listenEvents watches resource updates and consume those events
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing resource event source...")
	var resourceEventSource *v1alpha1.ResourceEventSource
	if err := yaml.Unmarshal(eventSource.Value, &resourceEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
	}

	logger.Infoln("setting up a K8s client")
	client, err := dynamic.NewForConfig(listener.K8RestConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to set up a dynamic K8s client for the event source %s", eventSource.Name)
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
			return errors.Wrapf(err, "failed to create the label selector for the event source %s", eventSource.Name)
		}
		options.LabelSelector = sel.String()
	}

	if resourceEventSource.Filter != nil && resourceEventSource.Filter.Fields != nil {
		sel, err := FieldSelector(resourceEventSource.Filter.Fields)
		if err != nil {
			return errors.Wrapf(err, "failed to create the field selector for the event source %s", eventSource.Name)
		}
		options.FieldSelector = sel.String()
	}

	tweakListOptions := func(op *metav1.ListOptions) {
		*op = *options
	}

	logger.Infoln("setting up informer factory...")
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, 0, resourceEventSource.Namespace, tweakListOptions)

	informer := factory.ForResource(gvr)

	informerEventCh := make(chan *InformerEvent)
	stopCh := make(chan struct{})
	startTime := time.Now()

	go func() {
		logger.Infoln("listening to resource events...")
		for {
			select {
			case event := <-informerEventCh:
				objBody, err := json.Marshal(event.Obj)
				if err != nil {
					logger.WithError(err).Errorln("failed to marshal the resource, rejecting the event...")
					continue
				}

				eventData := &events.ResourceEventData{
					EventType: string(event.Type),
					Body:      (*json.RawMessage)(&objBody),
					Group:     resourceEventSource.Group,
					Version:   resourceEventSource.Version,
					Resource:  resourceEventSource.Resource,
				}
				eventBody, err := json.Marshal(eventData)
				if err != nil {
					logger.WithError(err).Errorln("failed to marshal the event. rejecting the event...")
					continue
				}
				if !passFilters(event, resourceEventSource.Filter, startTime, logger) {
					continue
				}
				channels.Data <- eventBody
			case <-stopCh:
				return
			}
		}
	}()

	handlerFuncs := cache.ResourceEventHandlerFuncs{}

	for _, eventType := range resourceEventSource.EventTypes {
		switch eventType {
		case v1alpha1.ADD:
			handlerFuncs.AddFunc = func(obj interface{}) {
				logger.Infoln("detected create event")
				informerEventCh <- &InformerEvent{
					Obj:  obj,
					Type: v1alpha1.ADD,
				}
			}
		case v1alpha1.UPDATE:
			handlerFuncs.UpdateFunc = func(oldObj, newObj interface{}) {
				logger.Infoln("detected update event")
				informerEventCh <- &InformerEvent{
					Obj:    newObj,
					OldObj: oldObj,
					Type:   v1alpha1.UPDATE,
				}
			}
		case v1alpha1.DELETE:
			handlerFuncs.DeleteFunc = func(obj interface{}) {
				logger.Infoln("detected delete event")
				informerEventCh <- &InformerEvent{
					Obj:  obj,
					Type: v1alpha1.DELETE,
				}
			}
		default:
			stopCh <- struct{}{}
			return errors.Errorf("unknown event type: %s", string(eventType))
		}
	}

	sharedInformer := informer.Informer()
	sharedInformer.AddEventHandler(handlerFuncs)

	doneCh := make(chan struct{})

	logger.Infoln("running informer...")
	sharedInformer.Run(doneCh)

	<-channels.Done
	doneCh <- struct{}{}
	stopCh <- struct{}{}

	logger.Infoln("event source is stopped")
	close(informerEventCh)

	return nil
}

// LabelReq returns label requirements
func LabelReq(sel v1alpha1.Selector) (*labels.Requirement, error) {
	op := selection.Equals
	if sel.Operation != "" {
		op = selection.Operator(sel.Operation)
	}
	req, err := labels.NewRequirement(sel.Key, op, []string{sel.Value})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// LabelSelector returns label selector for resource filtering
func LabelSelector(selectors []v1alpha1.Selector) (labels.Selector, error) {
	var labelRequirements []labels.Requirement
	for _, sel := range selectors {
		req, err := LabelReq(sel)
		if err != nil {
			return nil, err
		}
		labelRequirements = append(labelRequirements, *req)
	}
	return labels.NewSelector().Add(labelRequirements...), nil
}

// FieldSelector returns field selector for resource filtering
func FieldSelector(selectors []v1alpha1.Selector) (fields.Selector, error) {
	var result []fields.Selector
	for _, sel := range selectors {
		op := selection.Equals
		if sel.Operation != "" {
			op = selection.Operator(sel.Operation)
		}
		selector, err := fields.ParseSelector(fmt.Sprintf("%s%s%s", sel.Key, op, sel.Value))
		if err != nil {
			return nil, err
		}
		result = append(result, selector)
	}
	return fields.AndSelectors(result...), nil
}

// helper method to check if the object passed the user defined filters
func passFilters(event *InformerEvent, filter *v1alpha1.ResourceFilter, startTime time.Time, log *logrus.Entry) bool {
	// no filters are applied.
	if filter == nil {
		return true
	}
	uObj := event.Obj.(*unstructured.Unstructured)
	if len(filter.Prefix) > 0 && !strings.HasPrefix(uObj.GetName(), filter.Prefix) {
		log.Infof("resource name does not match prefix. resource-name: %s, prefix: %s\n", uObj.GetName(), filter.Prefix)
		return false
	}
	created := uObj.GetCreationTimestamp()
	if !filter.CreatedBy.IsZero() && created.UTC().After(filter.CreatedBy.UTC()) {
		log.Infof("resource is created after filter time. creation-timestamp: %s, filter-creation-timestamp: %s\n", created.UTC().String(), filter.CreatedBy.UTC().String())
		return false
	}
	if filter.AfterStart && created.UTC().Before(startTime.UTC()) {
		log.Infof("resource is created before service start time. creation-timestamp: %s, start-timestamp: %s\n", created.UTC().String(), startTime.UTC().String())
		return false
	}
	return true
}
