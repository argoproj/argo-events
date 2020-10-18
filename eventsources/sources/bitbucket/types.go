package bitbucket

import (
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	gb "github.com/ktrysmt/go-bitbucket"
	"time"
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
	client               *gb.Client
	repoUuid             string
	bitbucketEventSource v1alpha1.BitbucketEventSource
}

// A paginated list of webhook subscriptions
type PaginatedWebhookSubscriptions struct {
	// Total number of objects in the response. This is an optional element that is not provided in all responses, as it can be expensive to compute.
	Size int32 `json:"size,omitempty"`
	// Page number of the current results. This is an optional element that is not provided in all responses.
	Page int32 `json:"page,omitempty"`
	// Current number of objects on the existing page. The default value is 10 with 100 being the maximum allowed value. Individual APIs may enforce different values.
	Pagelen int32 `json:"pagelen,omitempty"`
	// Link to the next page if it exists. The last page of a collection does not have this value. Use this link to navigate the result set and refrain from constructing your own URLs.
	Next string `json:"next,omitempty"`
	// Link to previous page if it exists. A collections first page does not have this value. This is an optional element that is not provided in all responses. Some result sets strictly support forward navigation and never provide previous links. Clients must anticipate that backwards navigation is not always available. Use this link to navigate the result set and refrain from constructing your own URLs.
	Previous string                `json:"previous,omitempty"`
	Values   []WebhookSubscription `json:"values,omitempty"`
}

type WebhookSubscription struct {
	Type_ string `json:"type"`
	// The webhook's id
	Uuid string `json:"uuid,omitempty"`
	// The URL events get delivered to.
	Url string `json:"url,omitempty"`
	// A user-defined description of the webhook.
	Description string `json:"description,omitempty"`
	// The type of entity, which is `repository` in the case of webhook subscriptions on repositories.
	SubjectType string       `json:"subject_type,omitempty"`
	Subject     *interface{} `json:"subject,omitempty"`
	Active      bool         `json:"active,omitempty"`
	CreatedAt   time.Time    `json:"created_at,omitempty"`
	// The events this webhook is subscribed to.
	Events []string `json:"events,omitempty"`
}
