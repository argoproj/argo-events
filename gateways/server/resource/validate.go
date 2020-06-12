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
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gatewayv1alpha1 "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"

	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/selection"
)

// ValidateEventSource validates a resource event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	if gatewayv1alpha1.EventSourceType(eventSource.Type) != gatewayv1alpha1.ResourceEvent {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  common.ErrEventSourceTypeMismatch(string(gatewayv1alpha1.ResourceEvent)),
		}, nil
	}

	var resourceEventSource *v1alpha1.ResourceEventSource
	if err := yaml.Unmarshal(eventSource.Value, &resourceEventSource); err != nil {
		listener.Logger.WithError(err).Errorln("failed to parse the event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, err
	}

	if err := validate(resourceEventSource); err != nil {
		listener.Logger.WithError(err).Errorln("failed to validate the event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, err
	}

	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func validate(eventSource *v1alpha1.ResourceEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.Version == "" {
		return fmt.Errorf("version must be specified")
	}
	if eventSource.Resource == "" {
		return fmt.Errorf("resource must be specified")
	}
	if eventSource.EventTypes == nil {
		return fmt.Errorf("event types must be specified")
	}
	if eventSource.Filter != nil {
		if eventSource.Filter.Labels != nil {
			if err := validateSelectors(eventSource.Filter.Labels); err != nil {
				return err
			}
		}
		if eventSource.Filter.Fields != nil {
			if err := validateSelectors(eventSource.Filter.Fields); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSelectors(selectors []v1alpha1.Selector) error {
	for _, sel := range selectors {
		if sel.Key == "" {
			return fmt.Errorf("key can't be empty for selector")
		}
		if sel.Operation == "" {
			continue
		}
		if selection.Operator(sel.Operation) == "" {
			return fmt.Errorf("unknown selection operation %s", sel.Operation)
		}
	}
	return nil
}
