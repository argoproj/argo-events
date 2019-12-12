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

package kafka

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
)

// ValidateEventSource validates the gateway event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	if apicommon.EventSourceType(eventSource.Type) != apicommon.KafkaEvent {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  common.ErrEventSourceTypeMismatch(string(apicommon.KafkaEvent)),
		}, nil
	}

	var kafkaGridEventSource *v1alpha1.KafkaEventSource
	if err := yaml.Unmarshal(eventSource.Value, &kafkaGridEventSource); err != nil {
		listener.Logger.WithError(err).Error("failed to parse the event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	if err := validate(kafkaGridEventSource); err != nil {
		listener.Logger.WithError(err).Error("failed to validate kafka event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func validate(eventSource *v1alpha1.KafkaEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.URL == "" {
		return fmt.Errorf("url must be specified")
	}
	if eventSource.Topic == "" {
		return fmt.Errorf("topic must be specified")
	}
	if eventSource.Partition == "" {
		return fmt.Errorf("partition must be specified")
	}
	return nil
}
