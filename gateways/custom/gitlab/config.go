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
	"github.com/ghodss/yaml"
	"github.com/xanzy/go-gitlab"
)

// GitlabEvent is the type of gitlab event to listen to
type GitlabEvent string

var (
	PushEvents  GitlabEvent             = "PushEvents"
	IssuesEvents   GitlabEvent          = "IssuesEvents"
	ConfidentialIssuesEvents GitlabEvent = "ConfidentialIssuesEvents"
	MergeRequestsEvents GitlabEvent     = "MergeRequestsEvents"
	TagPushEvents      GitlabEvent      = "TagPushEvents"
	NoteEvents        GitlabEvent       = "NoteEvents"
	JobEvents         GitlabEvent       = "JobEvents"
	PipelineEvents   GitlabEvent        = "PipelineEvents"
	WikiPageEvents    GitlabEvent       = "WikiPageEvents"
)

// GitlabExecutor implements ConfigExecutor
type GitlabExecutor struct {
	*gateways.GatewayConfig
	GitlabClient *gitlab.Client
}

// GitlabConfig contains information to setup a gitlab project integration
// +k8s:openapi-gen=true
type GitlabConfig struct {
	// ProjectId is the id of project for which integration needs to setup
	ProjectId string `json:"projectId"`

	// URL of a http server which is listening for gitlab events.
	// Refer webhook gateway for more details. https://github.com/argoproj/argo-events/blob/master/docs/tutorial.md#webhook
	URL string `json:"url"`

	// Event is a gitlab event to listen to.
	// Refer for supported events.
	Event GitlabEvent `json:"event"`

	// AccessToken is reference to k8 secret which holds the gitlab api access information
	AccessToken *GitlabSecret `json:"accessToken"`

	// EnableSSLVerification to enable ssl verification
	EnableSSLVerification bool `json:"enableSSLVerification"`

	// GitlabBaseURL is the base URL for API requests to a custom endpoint
	GitlabBaseURL string `json:"gitlabBaseUrl"`
}

// GitlabSecret contains information of k8 secret which holds the gitlab api access information
// +k8s:openapi-gen=true
type GitlabSecret struct {
	// Key within the K8 secret for access token
	Key string
	// Name of K8 secret containing access token info
	Name string
}

// cred stores the api access token
type cred struct {
	// token is gitlab api access token
	token string
}

// parseConfig parses a configuration of gateway
func parseConfig(config string) (*GitlabConfig, error) {
	var g *GitlabConfig
	err := yaml.Unmarshal([]byte(config), &g)
	if err != nil {
		return nil, err
	}
	return g, err
}
