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

package stripe

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
)

// ValidateEventSource validates stripe event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.StripeEventSource)
}

func validate(eventSource *v1alpha1.StripeEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.CreateWebhook {
		if eventSource.APIKey == nil {
			return fmt.Errorf("api key K8s secret selector not provided")
		}
	}
	return webhook.ValidateWebhookContext(eventSource.Webhook)
}
