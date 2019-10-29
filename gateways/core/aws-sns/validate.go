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

package aws_sns

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"

	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates gateway event source
func (listener *SNSEventSourceListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	var snsEventSource *v1alpha1.SNSEventSource
	if err := yaml.Unmarshal(eventSource.Value, &snsEventSource); err != nil {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  err.Error(),
		}, err
	}

	if err := validateSNSEventSource(snsEventSource); err != nil {
		return &gateways.ValidEventSource{
			Reason:  err.Error(),
			IsValid: false,
		}, err
	}

	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

// validateSNSEventSource checks if sns event source is valid
func validateSNSEventSource(snsEventSource *v1alpha1.SNSEventSource) error {
	if snsEventSource == nil {
		return gwcommon.ErrNilEventSource
	}
	if snsEventSource.TopicArn == "" {
		return fmt.Errorf("must specify topic arn")
	}
	if snsEventSource.Region == "" {
		return fmt.Errorf("must specify region")
	}
	return gwcommon.ValidateWebhook(snsEventSource.WebHook)
}
