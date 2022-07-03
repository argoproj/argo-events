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

package bitbucket

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates bitbucketserver event source
func (el *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&el.BitbucketEventSource)
}

func validate(eventSource *v1alpha1.BitbucketEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.ProjectKey == "" {
		return fmt.Errorf("project key can't be empty")
	}
	if eventSource.RepositorySlug == "" {
		return fmt.Errorf("repository slug can't be empty")
	}
	if eventSource.Owner == "" {
		return fmt.Errorf("owner can't be empty")
	}
	if eventSource.ShouldCreateWebhook() && len(eventSource.Events) == 0 {
		return fmt.Errorf("events must be defined to create a bitbucket webhook")
	}
	return webhook.ValidateWebhookContext(eventSource.Webhook)
}
