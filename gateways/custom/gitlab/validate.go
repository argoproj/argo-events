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
)

// ValidateEventSource validates gitlab gateway event source
func (ese *GitlabEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	g, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}
	if g == nil {
		return v, gateways.ErrEmptyEventSource
	}
	if g.ProjectId == "" {
		return v, fmt.Errorf("project id can't be empty")
	}
	if g.Event == "" {
		return v, fmt.Errorf("event type can't be empty")
	}
	if g.URL == "" {
		return v, fmt.Errorf("url can't be empty")
	}
	if g.GitlabBaseURL == "" {
		return v, fmt.Errorf("gitlab base url can't be empty")
	}
	if g.AccessToken == nil {
		return v, fmt.Errorf("access token can't be nil")
	}
	return v, nil
}
