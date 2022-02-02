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

package bitbucketserver

import (
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements ConfigExecutor
type EventListener struct {
	EventSourceName            string
	EventName                  string
	BitbucketServerEventSource v1alpha1.BitbucketServerEventSource
	Metrics                    *metrics.Metrics
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
	return apicommon.BitbucketServerEvent
}

// Router contains the configuration information for a route
type Router struct {
	// route contains information about a API endpoint
	route *webhook.Route
	// hookID is bitbucket Webhook ID
	// Bitbucket Server API docs:
	// https://developer.atlassian.com/server/bitbucket/reference/rest-api/
	hookID int
	// bitbucketserverEventSource is the event source that contains configuration necessary to consume events from Bitbucket Server
	bitbucketserverEventSource *v1alpha1.BitbucketServerEventSource
}
