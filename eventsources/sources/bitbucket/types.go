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

package bitbucket

import (
	bitbucketv2 "github.com/ktrysmt/go-bitbucket"

	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements ConfigExecutor
type EventListener struct {
	EventSourceName      string
	EventName            string
	BitbucketEventSource v1alpha1.BitbucketEventSource
	Metrics              *metrics.Metrics
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
	return apicommon.BitbucketEvent
}

// Router contains the configuration information for a route
type Router struct {
	// route contains information about a API endpoint
	route *webhook.Route
	// client to connect to Bitbucket
	client *bitbucketv2.Client
	// hookID holds the the uuid of the webhook that is created in Bitbucket
	hookID string
	// bitbucketEventSource is the event source that holds information to consume events from Bitbucket
	bitbucketEventSource *v1alpha1.BitbucketEventSource
}

type WebhookSubscription struct {
	// Uuid holds the webhook's ID
	Uuid string `json:"uuid,omitempty"`
	// The Url events get delivered to.
	Url string `json:"url,omitempty"`
	// Description holds a user-defined description of the webhook.
	Description string `json:"description,omitempty"`
	// Subject holds metadata about the subject of the webhook (repository, etc.)
	Subject *interface{} `json:"subject,omitempty"`
	// Active refers to status of the webhook for event deliveries.
	Active bool `json:"active,omitempty"`
	// The Events this webhook is subscribed to.
	Events []string `json:"events,omitempty"`
}

// AuthStrategy is implemented by the different Bitbucket auth strategies that are supported
type AuthStrategy interface {
	// BitbucketClient returns a bitbucket client initialized with the specific auth strategy
	BitbucketClient() *bitbucketv2.Client
}
