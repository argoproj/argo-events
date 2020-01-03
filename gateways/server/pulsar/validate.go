/*
Copyright 2019 Lucidworks Inc.

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

package pulsar

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

// validates Pulsar event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {

	logrus.WithField("eventSource", eventSource).Info(">> ValidateEventSource") // todo debug

	if apicommon.EventSourceType(eventSource.Type) != apicommon.PulsarEvent {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  common.ErrEventSourceTypeMismatch(string(apicommon.PulsarEvent)),
		}, nil
	}

	var pulsarEventSource *v1alpha1.PulsarEventSource
	if err := yaml.Unmarshal(eventSource.Value, &pulsarEventSource); err != nil {
		logrus.WithError(err).Error("Cannot validate the event source: cannot unmarshal the YAML")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	if err := validate(pulsarEventSource); err != nil {
		logrus.WithError(err).Error("Cannot validate the event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	logrus.WithField("validatedSource", eventSource).Info("Pulsar event source validated successfully") // todo debug
	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func validate(eventSource *v1alpha1.PulsarEventSource) error {
	logrus.WithField("eventSource", eventSource).Info(">> validatePulsarConfig") // todo debug

	if eventSource == nil {
		return common.ErrNilEventSource
	}

	if eventSource.PulsarConfigUrl == "" {
		return fmt.Errorf("The Pulsar service URL must be specified")
	}
	if eventSource.PulsarConfigSubscriptionType == "" {
		return fmt.Errorf("The Pulsar Subscription Type must be specified")
	}

	logrus.WithField("eventSource", eventSource).Info("<< validatePulsarConfig") // todo debug

	return nil
}
