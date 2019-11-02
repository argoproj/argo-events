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

package amqp

import (
	"context"
	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
)

// ValidateEventSource validates gateway event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	var amqpEventSource *v1alpha1.AMQPEventSource
	if err := yaml.Unmarshal(eventSource.Value, &amqpEventSource); err != nil {
		listener.Logger.WithError(err).Error("failed to parse the event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	if err := validate(amqpEventSource); err != nil {
		listener.Logger.WithError(err).Error("failed to validate amqp event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func validate(eventSource *v1alpha1.AMQPEventSource) error {
	if eventSource == nil {
		return gwcommon.ErrNilEventSource
	}
	if eventSource.URL == "" {
		return errors.New("url must be specified")
	}
	if eventSource.RoutingKey == "" {
		return errors.New("routing key must be specified")
	}
	if eventSource.ExchangeName == "" {
		return errors.New("exchange name must be specified")
	}
	if eventSource.ExchangeType == "" {
		return errors.New("exchange type must be specified")
	}
	return nil
}
