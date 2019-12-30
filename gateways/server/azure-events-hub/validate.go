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

package azure_events_hub

import (
	"context"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// ValidateEventSource validates azure events hub event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	if apicommon.EventSourceType(eventSource.Type) != apicommon.AzureEventsHub {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  common.ErrEventSourceTypeMismatch(string(apicommon.AzureEventsHub)),
		}, nil
	}

	var azureEventsHubSource *v1alpha1.AzureEventsHubEventSource
	if err := yaml.Unmarshal(eventSource.Value, &azureEventsHubSource); err != nil {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	if err := validate(azureEventsHubSource); err != nil {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, nil
	}

	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func validate(eventSource *v1alpha1.AzureEventsHubEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.FQDN == "" {
		return errors.New("FQDN is not specified")
	}
	if eventSource.HubName == "" {
		return errors.New("hub name/path is not specified")
	}
	if eventSource.SharedAccessKey == nil {
		return errors.New("SharedAccessKey is not specified")
	}
	if eventSource.SharedAccessKeyName == nil {
		return errors.New("SharedAccessKeyName is not specified")
	}
	if eventSource.Namespace == "" {
		return errors.New("namespace to retrieve the SharedAccessKey and SharedAccessKeyName secret info is not specified")
	}
	return nil
}
