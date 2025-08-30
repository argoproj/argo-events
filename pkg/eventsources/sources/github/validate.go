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

package github

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
)

// ValidateEventSource validates a github event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.GithubEventSource)
}

func validate(githubEventSource *v1alpha1.GithubEventSource) error {
	if githubEventSource == nil {
		return v1alpha1.ErrNilEventSource
	}
	if githubEventSource.GetOwnedRepositories() == nil && githubEventSource.Organizations == nil {
		return fmt.Errorf("either repositories or organizations is required")
	}
	if githubEventSource.GetOwnedRepositories() != nil && githubEventSource.Organizations != nil {
		return fmt.Errorf("only one of repositories and organizations is allowed")
	}
	if githubEventSource.NeedToCreateHooks() && len(githubEventSource.Events) == 0 {
		return fmt.Errorf("events must be defined to create a github webhook")
	}

	if githubEventSource.ContentType != "" {
		if !(githubEventSource.ContentType == "json" || githubEventSource.ContentType == "form") { //nolint:staticcheck
			return fmt.Errorf("content type must be \"json\" or \"form\"")
		}
	}

	// in order to avoid requests ending accidentally to public GitHub,
	// make sure that both are set if either one is provided
	if githubEventSource.GithubBaseURL != "" || githubEventSource.GithubUploadURL != "" { //nolint:staticcheck
		if githubEventSource.GithubBaseURL == "" {
			return fmt.Errorf("githubBaseURL is required when githubUploadURL is set")
		}
		if githubEventSource.GithubUploadURL == "" {
			return fmt.Errorf("githubUploadURL is required when githubBaseURL is set")
		}
	}
	return webhook.ValidateWebhookContext(githubEventSource.Webhook)
}
