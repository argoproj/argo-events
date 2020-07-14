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

package storagegrid

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates storage grid event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.StorageGridEventSource)
}

func validate(eventSource *v1alpha1.StorageGridEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.TopicArn == "" {
		return fmt.Errorf("topic arn must be provided")
	}
	if eventSource.APIURL == "" {
		return fmt.Errorf("api url must be provided")
	}
	if eventSource.AuthToken == nil {
		return fmt.Errorf("api auth token must be provided")
	}
	if eventSource.Bucket == "" {
		return fmt.Errorf("bucket name must be provided")
	}
	if eventSource.Events == nil {
		return fmt.Errorf("event types must be provided")
	}
	return webhook.ValidateWebhookContext(eventSource.Webhook)
}
