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

package pubsub

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
)

// ValidateEventSource validates gateway event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	if apicommon.EventSourceType(eventSource.Type) != apicommon.PubSubEvent {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  common.ErrEventSourceTypeMismatch(string(apicommon.PubSubEvent)),
		}, nil
	}

	var pubsubEventSource *v1alpha1.PubSubEventSource
	if err := yaml.Unmarshal(eventSource.Value, &pubsubEventSource); err != nil {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	if err := validate(pubsubEventSource); err != nil {
		return &gateways.ValidEventSource{
			Reason:  err.Error(),
			IsValid: false,
		}, nil
	}

	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func validate(eventSource *v1alpha1.PubSubEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.ProjectID == "" {
		return fmt.Errorf("must specify projectId")
	}
	if eventSource.Topic == "" {
		return fmt.Errorf("must specify topic")
	}
	if eventSource.CredentialsFile == "" {
		return fmt.Errorf("must specify credentials file path")
	}
	return nil
}
