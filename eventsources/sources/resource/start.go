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
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// InformerEvent holds event generated from resource state change
type InformerEvent struct {
	Obj    interface{}
	OldObj interface{}
	Type   v1alpha1.ResourceEventType
}

// EventListener implements Eventing
type EventListener struct {
	EventSourceName     string
	EventName           string
	ResourceEventSource v1alpha1.ResourceEventSource
	Metrics             *metrics.Metrics
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.ResourceEvent
}

// StartListening watches resource updates and consume those events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	log.Info("setting up a K8s client")
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to get a K8s rest config for the event source %s", el.GetEventName())
	}
	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to set up a dynamic K8s client for the event source %s", el.GetEventName())
	}

	resourceEventSource := &el.ResourceEventSource

	gvr := schema.GroupVersionResource{
		Group:    resourceEventSource.Group,
		Version:  resourceEventSource.Version,
		Resource: resourceEventSource.Resource,
	}

	client.Resource(gvr)

	options := &metav1.ListOptions{}

	log.Info("configuring label selectors if filters are selected...")
	if resourceEventSource.Filter != nil && resourceEventSource.Filter.Labels != nil {
		sel, err := LabelSelector(resourceEventSource.Filter.Labels)
		if err != nil {
			return errors.Wrapf(err, "failed to create the label selector for the event source %s", el.GetEventName())
		}
		options.LabelSelector = sel.String()
	}

	tweakListOptions := func(op *metav1.ListOptions) {
		*op = *options
	}

	log.Info("setting up informer factory...")
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, 0, resourceEventSource.Namespace, tweakListOptions)

	informer := factory.ForResource(gvr)

	informerEventCh := make(chan *InformerEvent)
	stopCh := make(chan struct{})
	startTime := time.Now()

	processOne := func(event *InformerEvent) error {
		if !passFilters(event, resourceEventSource.Filter, startTime, log) {
			return nil
		}

		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		objBody, err := json.Marshal(event.Obj)
		if err != nil {
			return errors.Wrap(err, "failed to marshal the resource, rejecting the event...")
		}

		var oldObjBody []byte
		if event.OldObj != nil {
			oldObjBody, err = json.Marshal(event.OldObj)
			if err != nil {
				return errors.Wrap(err, "failed to marshal the resource, rejecting the event...")
			}
		}

		eventData := &events.ResourceEventData{
			EventType: string(event.Type),
			Body:      (*json.RawMessage)(&objBody),
			OldBody:   (*json.RawMessage)(&oldObjBody),
			Group:     resourceEventSource.Group,
			Version:   resourceEventSource.Version,
			Resource:  resourceEventSource.Resource,
			Metadata:  resourceEventSource.Metadata,
		}

		eventBody, err := json.Marshal(eventData)
		if err != nil {
			return errors.Wrap(err, "failed to marshal the event. rejecting the event...")
		}

		if err = dispatch(eventBody); err != nil {
			return errors.Wrap(err, "failed to dispatch a resource event")
		}
		return nil
	}

	go func() {
		log.Info("listening to resource events...")
		for {
			select {
			case event := <-informerEventCh:
				if err := processOne(event); err != nil {
					log.Errorw("failed to process a Resource event", zap.Error(err))
					el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
				}
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
				log.Info("detected create event")
				informerEventCh <- &InformerEvent{
					Obj:  obj,
					Type: v1alpha1.ADD,
				}
			}
		case v1alpha1.UPDATE:
			handlerFuncs.UpdateFunc = func(oldObj, newObj interface{}) {
				log.Info("detected update event")
				informerEventCh <- &InformerEvent{
					Obj:    newObj,
					OldObj: oldObj,
					Type:   v1alpha1.UPDATE,
				}
			}
		case v1alpha1.DELETE:
			handlerFuncs.DeleteFunc = func(obj interface{}) {
				log.Info("detected delete event")
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

	log.Info("running informer...")
	sharedInformer.Run(doneCh)

	<-ctx.Done()
	doneCh <- struct{}{}
	stopCh <- struct{}{}

	log.Info("event source is stopped")
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

// helper method to check if the object passed the user defined filters
func passFilters(event *InformerEvent, filter *v1alpha1.ResourceFilter, startTime time.Time, log *zap.SugaredLogger) bool {
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
	if len(filter.Fields) > 0 {
		jsData, err := uObj.MarshalJSON()
		if err != nil {
			log.Errorw("failed to marshal informer event", zap.Error(err))
			return false
		}

		return filterFields(jsData, filter.Fields, log)
	}
	return true
}

func filterFields(jsonData []byte, selectors []v1alpha1.Selector, log *zap.SugaredLogger) bool {
	for _, selector := range selectors {
		res := gjson.GetBytes(jsonData, selector.Key)
		if !res.Exists() {
			return false
		}
		exp, err := regexp.Compile(selector.Value)
		if err != nil {
			log.Errorw("invalid regex", zap.Error(err))
			return false
		}
		match := exp.Match([]byte(res.Str))

		switch selection.Operator(selector.Operation) {
		case selection.Equals, selection.DoubleEquals:
			if !match {
				return false
			}
		case selection.NotEquals:
			if match {
				return false
			}
		default:
			log.Errorf("invalid operator, only %v, %v and %v are supported", selection.Equals, selection.DoubleEquals, selection.NotEquals)
			return false
		}
	}
	return true
}
