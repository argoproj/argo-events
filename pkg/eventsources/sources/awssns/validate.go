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

package awssns

import (
	"context"
	"fmt"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// ValidateEventSource validates sns event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.SNSEventSource)
}

func validate(snsEventSource *aev1.SNSEventSource) error {
	if snsEventSource == nil {
		return sharedutil.ErrNilEventSource
	}
	if snsEventSource.TopicArn == "" {
		return fmt.Errorf("must specify topic arn")
	}
	if snsEventSource.Region == "" {
		return fmt.Errorf("must specify region")
	}
	return webhook.ValidateWebhookContext(snsEventSource.Webhook)
}
