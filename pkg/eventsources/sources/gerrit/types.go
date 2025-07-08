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

package gerrit

import (
	gerrit "github.com/andygrunwald/go-gerrit"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/metrics"
)

// EventListener implements ConfigExecutor
type EventListener struct {
	EventSourceName   string
	EventName         string
	GerritEventSource v1alpha1.GerritEventSource
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
	return v1alpha1.GerritEvent
}

// Router contains the configuration information for a route
type Router struct {
	// route contains information about a API endpoint
	route *webhook.Route
	// gerritClient is the client to connect to Gerrit
	gerritClient *gerrit.Client
	// gerritClient is the client to connect to Gerrit
	gerritHookService *gerritWebhookService
	// project -> hook
	projectHooks map[string]string
	// gerritEventSource is the event source that contains configuration necessary to consume events from Gerrit
	gerritEventSource *v1alpha1.GerritEventSource
}

// ProjectHookConfigs is the config for gerrit project
// Ref: https://gerrit.googlesource.com/plugins/webhooks/+doc/master/src/main/resources/Documentation/config.md
type ProjectHookConfigs struct {
	// URL: Address of the remote server to post events to
	URL string `json:"url,omitempty"`
	// Events:
	// Type of the event which will be posted to the remote url. Multiple event types can be specified, listing event types which should be posted.
	// When no event type is configured, all events will be posted.
	Events []string `json:"events,omitempty"`
	// ConnectionTimeout:
	// Maximum interval of time in milliseconds the plugin waits for a connection to the target instance.
	// When not specified, the default value is derived from global configuration.
	ConnectionTimeout string `json:"connectionTimeout,omitempty"`
	// SocketTimeout:
	// Maximum interval of time in milliseconds the plugin waits for a response from the target instance once the connection has been established.
	// When not specified, the default value is derived from global configuration.
	SocketTimeout string `json:"socketTimeout,omitempty"`
	// MaxTries:
	// Maximum number of times the plugin should attempt when posting an event to the target url. Setting this value to 0 will disable retries.
	// When not specified, the default value is derived from global configuration.
	MaxTries string `json:"maxTries,omitempty"`
	// RetryInterval:
	// The interval of time in milliseconds between the subsequent auto-retries.
	// When not specified, the default value is derived from global configuration.
	RetryInterval string `json:"retryInterval,omitempty"`
	// SslVerify:
	// When 'true' SSL certificate verification of remote url is performed when payload is delivered, the default value is derived from global configuration.
	SslVerify bool `json:"sslVerify,omitempty"`
}
