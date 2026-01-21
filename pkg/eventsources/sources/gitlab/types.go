/*
Copyright 2018 The Argoproj Authors.

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
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
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
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.GitlabEvent
}

// Router contains the configuration information for a route
type Router struct {
	// route contains information about a API endpoint
	route *webhook.Route
	// gitlabClient is the client to connect to GitLab
	gitlabClient *gitlab.Client
	// projectID -> hook ID
	projectHookIDs map[string]int64
	// groupID -> hook ID
	groupHookIDs map[string]int64
	// gitlabEventSource is the event source that contains configuration necessary to consume events from GitLab
	gitlabEventSource *v1alpha1.GitlabEventSource
	// gitlab webhook secret token
	secretToken string
}
