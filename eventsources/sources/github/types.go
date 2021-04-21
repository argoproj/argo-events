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
	"github.com/google/go-github/v31/github"

	"github.com/argoproj/argo-events/eventsources/common/webhook"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for GitHub event source
type EventListener struct {
	EventSourceName   string
	EventName         string
	GithubEventSource v1alpha1.GithubEventSource
	Metrics           *metrics.Metrics
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.GithubEvent
}

// Router contains information about the route
type Router struct {
	// route contains configuration for an API endpoint
	route *webhook.Route
	// githubEventSource is the event source that holds information to consume events from GitHub
	githubEventSource *v1alpha1.GithubEventSource
	// githubClient is the client to connect to GitHub
	githubClient *github.Client
	// hookIDs represents GitHub (web and service) hook ID for repositories.
	// The map key is the combo of owner and repo name
	hookIDs map[string]int64
	// hookSecret is a GitHub webhook secret
	hookSecret string
}

// cred stores the api access token or webhook secret
type cred struct {
	secret string
}
