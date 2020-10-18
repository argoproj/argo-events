package bitbucket

import (
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	gb "github.com/ktrysmt/go-bitbucket"
)

// EventListener implements ConfigExecutor
type EventListener struct {
	EventSourceName      string
	EventName            string
	BitbucketEventSource v1alpha1.BitbucketEventSource
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
	// client
	client *gb.Client
}
