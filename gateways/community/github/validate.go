/*
Copyright 2018 KompiTech GmbH
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
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// Validate validates github gateway configuration
func (ese *GithubEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(es.Data, parseEventSource, validateGithub)
}

func validateGithub(config interface{}) error {
	g := config.(*githubConfig)
	if g == nil {
		return gwcommon.ErrNilEventSource
	}
	if g.Repository == "" {
		return fmt.Errorf("repository cannot be empty")
	}
	if g.Owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if g.APIToken == nil {
		return fmt.Errorf("api token can't be empty")
	}
	if g.Events == nil || len(g.Events) < 1 {
		return fmt.Errorf("events must be defined")
	}
	if g.ContentType != "" {
		if !(g.ContentType == "json" || g.ContentType == "form") {
			return fmt.Errorf("content type must be \"json\" or \"form\"")
		}
	}
	return gwcommon.ValidateWebhook(g.Hook)
}
