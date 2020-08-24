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

package gcppubsub

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates gateway event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.PubSubEventSource)
}

func validate(eventSource *v1alpha1.PubSubEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.TopicProjectID != "" && eventSource.Topic == "" {
		return fmt.Errorf("you can't specify topicProjectID if you don't specify topic")
	}
	if eventSource.Topic == "" && eventSource.SubscriptionID == "" {
		return fmt.Errorf("must specify topic or subscriptionID")
	}
	return nil
}
