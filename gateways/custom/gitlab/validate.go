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
	"github.com/argoproj/argo-events/gateways"
	"fmt"
)

func (ce *GitlabExecutor) Validate(config *gateways.ConfigContext) error {
	g, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if g == nil {
		return gateways.ErrEmptyConfig
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
	return nil
}
