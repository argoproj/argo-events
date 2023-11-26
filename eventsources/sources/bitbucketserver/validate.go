/*

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

package bitbucketserver

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates bitbucketserver event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.BitbucketServerEventSource)
}

func validate(eventSource *v1alpha1.BitbucketServerEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.GetBitbucketServerRepositories() == nil && len(eventSource.Projects) == 0 {
		return fmt.Errorf("at least one project or repository configuration is required")
	}
	if eventSource.ShouldCreateWebhooks() && len(eventSource.Events) == 0 {
		return fmt.Errorf("events must be defined to create a bitbucket server webhook")
	}
	if eventSource.BitbucketServerBaseURL == "" {
		return fmt.Errorf("bitbucket server base url can't be empty")
	}
	if eventSource.CheckInterval != "" {
		if _, err := time.ParseDuration(eventSource.CheckInterval); err != nil {
			return fmt.Errorf("failed to parse webhook check interval duration: %w", err)
		}
	}
	return webhook.ValidateWebhookContext(eventSource.Webhook)
}
