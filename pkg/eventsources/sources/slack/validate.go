/*
Copyright 2018 The Argoproj Authors.

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

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
)

// ValidateEventSource validates slack event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.SlackEventSource)
}

func validate(eventSource *v1alpha1.SlackEventSource) error {
	if eventSource == nil {
		return v1alpha1.ErrNilEventSource
	}
	if eventSource.Token == nil {
		return fmt.Errorf("token not provided")
	}
	if eventSource.SigningSecret == nil {
		return fmt.Errorf("signingSecret not provided")
	}
	return webhook.ValidateWebhookContext(eventSource.Webhook)
}
