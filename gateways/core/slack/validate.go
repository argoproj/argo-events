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

package slack

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways/common/webhook"

	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
)

// ValidateEventSource validates slack event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	var slackEventSource *v1alpha1.SlackEventSource
	if err := yaml.Unmarshal(eventSource.Value, &slackEventSource); err != nil {
		listener.Logger.WithError(err).Errorln("failed to parse the event source")
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, err
	}

	if err := validate(slackEventSource); err != nil {
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

func validate(eventSource *v1alpha1.SlackEventSource) error {
	if eventSource == nil {
		return gwcommon.ErrNilEventSource
	}
	if eventSource.Token == nil {
		return fmt.Errorf("token not provided")
	}
	return webhook.ValidateWebhookContext(eventSource.WebHook)
}
