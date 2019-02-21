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

package gitlab

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates gitlab gateway event source
func (ese *GitlabEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	g, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}
	if err != nil {
		gateways.SetValidEventSource(v, fmt.Sprintf("%s. err: %s", gateways.ErrEventSourceParseFailed, err.Error()), false)
		return v, nil
	}
	if err = validateGitlab(g); err != nil {
		gateways.SetValidEventSource(v, err.Error(), false)
		return v, gateways.ErrInvalidEventSource
	}
	gateways.SetValidEventSource(v, "", true)
	return v, nil
}

func validateGitlab(g *glab) error {
	if g == nil {
		return gateways.ErrEmptyEventSource
	}
	if g.ProjectId == "" {
		return fmt.Errorf("project id can't be empty")
	}
	if g.Event == "" {
		return fmt.Errorf("event type can't be empty")
	}
	if g.URL == "" {
		return fmt.Errorf("url can't be empty")
	}
	if g.GitlabBaseURL == "" {
		return fmt.Errorf("gitlab base url can't be empty")
	}
	if g.AccessToken == nil {
		return fmt.Errorf("access token can't be nil")
	}
	return gwcommon.ValidateWebhook(g.Endpoint, g.Port)
}
