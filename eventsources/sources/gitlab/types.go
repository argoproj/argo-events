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
	"github.com/xanzy/go-gitlab"

	"github.com/argoproj/argo-events/eventsources/common/webhook"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements ConfigExecutor
type EventListener struct {
	EventSourceName   string
	EventName         string
	GitlabEventSource v1alpha1.GitlabEventSource
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
	return apicommon.GitlabEvent
}

// Router contains the configuration information for a route
type Router struct {
	// route contains information about a API endpoint
	route *webhook.Route
	// gitlabClient is the client to connect to GitLab
	gitlabClient *gitlab.Client
	// projectID -> hook ID
	hookIDs map[string]int
	// gitlabEventSource is the event source that contains configuration necessary to consume events from GitLab
	gitlabEventSource *v1alpha1.GitlabEventSource
	// gitlab webhook secret token
	secretToken string
}

// cred stores the api access token
type cred struct {
	// token is gitlab api access token
	token string
}
