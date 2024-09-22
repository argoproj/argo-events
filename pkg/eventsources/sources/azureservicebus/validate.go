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

package azureservicebus

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

// ValidateEventSource validates azure events hub event source
func (el *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&el.AzureServiceBusEventSource)
}

func validate(eventSource *v1alpha1.AzureServiceBusEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.ConnectionString == nil && eventSource.FullyQualifiedNamespace == "" {
		return fmt.Errorf("ConnectionString or fullyQualifiedNamespace must be specified")
	}
	if eventSource.QueueName == "" && (eventSource.TopicName == "" || eventSource.SubscriptionName == "") {
		return fmt.Errorf("QueueName or TopicName/SubscriptionName must be specified")
	}
	if eventSource.QueueName != "" && (eventSource.TopicName != "" || eventSource.SubscriptionName != "") {
		return fmt.Errorf("QueueName and TopicName/SubscriptionName cannot be specified at the same time")
	}
	if eventSource.TopicName == "" && eventSource.SubscriptionName != "" {
		return fmt.Errorf("TopicName must be specified when SubscriptionName is specified")
	}
	if eventSource.TopicName != "" && eventSource.SubscriptionName == "" {
		return fmt.Errorf("SubscriptionName must be specified when TopicName is specified")
	}

	return nil
}
