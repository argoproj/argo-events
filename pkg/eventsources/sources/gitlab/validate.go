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

package gitlab

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
)

// ValidateEventSource validates gitlab event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.GitlabEventSource)
}

func validate(eventSource *v1alpha1.GitlabEventSource) error {
	if eventSource == nil {
		return v1alpha1.ErrNilEventSource
	}
	if eventSource.Events == nil {
		return fmt.Errorf("events can't be empty")
	}
	if eventSource.GitlabBaseURL == "" {
		return fmt.Errorf("gitlab base url can't be empty")
	}
	if eventSource.AccessToken == nil {
		return fmt.Errorf("access token can't be nil")
	}
	return webhook.ValidateWebhookContext(eventSource.Webhook)
}
