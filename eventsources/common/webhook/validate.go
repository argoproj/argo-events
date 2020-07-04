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

package webhook

import (
	"fmt"
	"strconv"

	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateWebhookContext validates a webhook context
func ValidateWebhookContext(context *v1alpha1.WebhookContext) error {
	if context == nil {
		return fmt.Errorf("")
	}
	if context.Endpoint == "" {
		return fmt.Errorf("endpoint can't be empty")
	}
	if context.Port == "" {
		return fmt.Errorf("port can't be empty")
	}
	if context.Port != "" {
		_, err := strconv.Atoi(context.Port)
		if err != nil {
			return fmt.Errorf("failed to parse server port %s. err: %+v", context.Port, err)
		}
	}
	return nil
}

// validateRoute validates a route
func validateRoute(r *Route) error {
	if r == nil {
		return fmt.Errorf("route can't be nil")
	}
	if r.Context == nil {
		return fmt.Errorf("webhook can't be nil")
	}
	if r.StartCh == nil {
		return fmt.Errorf("start channel can't be nil")
	}
	if r.EventSourceName == "" {
		return fmt.Errorf("event source name can't be empty")
	}
	if r.EventName == "" {
		return fmt.Errorf("event name can't be empty")
	}
	if r.Logger == nil {
		return fmt.Errorf("logger can't be nil")
	}
	return nil
}
