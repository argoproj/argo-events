/*
Copyright 2020 BlackRock, Inc.

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
package redis

import (
	"context"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// ValidateEventSource validates nats event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	if apicommon.EventSourceType(eventSource.Type) != apicommon.RedisEvent {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  common.ErrEventSourceTypeMismatch(string(apicommon.RedisEvent)),
		}, nil
	}

	var redisEventSource *v1alpha1.RedisEventSource
	if err := yaml.Unmarshal(eventSource.Value, &redisEventSource); err != nil {
		listener.Logger.WithError(err).Error("failed to parse the event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	if err := validate(redisEventSource); err != nil {
		listener.Logger.WithError(err).Error("failed to validate redis event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func validate(eventSource *v1alpha1.RedisEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.HostAddress == "" {
		return errors.New("host address must be specified")
	}
	if eventSource.Channels == nil {
		return errors.New("channel/s must be specified")
	}
	if eventSource.Password != nil && eventSource.Namespace == "" {
		return errors.New("namespace must be defined in order to retrieve the password from the secret")
	}
	return nil
}
