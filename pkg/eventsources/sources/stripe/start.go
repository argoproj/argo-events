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

package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhookendpoint"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// controller controls the webhook operations
var (
	controller = webhook.NewController()
)

// set up the activation and inactivation channels to control the state of routes.
func init() {
	go webhook.ProcessRouteStatus(controller)
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
	return v1alpha1.StripeEvent
}

// Implement Router
// 1. GetRoute
// 2. HandleRoute
// 3. PostActivate
// 4. PostDeactivate

// GetRoute returns the route
func (rc *Router) GetRoute() *webhook.Route {
	return rc.route
}

// HandleRoute handles incoming requests on the route
func (rc *Router) HandleRoute(writer http.ResponseWriter, request *http.Request) {
	route := rc.route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
	)

	logger.Info("request a received, processing it...")

	if !route.Active {
		logger.Warn("endpoint is not active, won't process it")
		sharedutil.SendErrorResponse(writer, "endpoint is inactive")
		return
	}

	defer func(start time.Time) {
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	request.Body = http.MaxBytesReader(writer, request.Body, route.Context.GetMaxPayloadSize())
	payload, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Errorw("error reading request body", zap.Error(err))
		writer.WriteHeader(http.StatusServiceUnavailable)
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	var event *stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		logger.Errorw("failed to parse request body", zap.Error(err))
		sharedutil.SendErrorResponse(writer, "failed to parse the event")
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	if ok := filterEvent(event, rc.stripeEventSource.EventFilter); !ok {
		logger.Errorw("failed to pass the filters", zap.Any("event-type", event.Type), zap.Error(err))
		sharedutil.SendErrorResponse(writer, "invalid event")
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	eventData := &events.StripeEventData{
		Event:    event,
		Metadata: rc.stripeEventSource.Metadata,
	}

	data, err := json.Marshal(eventData)
	if err != nil {
		logger.Errorw("failed to marshal event data", zap.Any("event-id", event.ID), zap.Error(err))
		sharedutil.SendErrorResponse(writer, "invalid event")
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	webhook.DispatchEvent(route, data, logger, writer)
}

// PostActivate performs operations once the route is activated and ready to consume requests
func (rc *Router) PostActivate() error {
	if rc.stripeEventSource.CreateWebhook {
		route := rc.route
		stripeEventSource := rc.stripeEventSource
		logger := route.Logger.With(
			logging.LabelEndpoint, route.Context.Endpoint,
			logging.LabelHTTPMethod, route.Context.Method,
		)
		logger.Info("registering a new webhook")

		apiKey, err := sharedutil.GetSecretFromVolume(stripeEventSource.APIKey)
		if err != nil {
			return fmt.Errorf("APIKey not found, %w", err)
		}

		stripe.Key = apiKey

		params := &stripe.WebhookEndpointParams{
			URL: stripe.String(sharedutil.FormattedURL(stripeEventSource.Webhook.URL, stripeEventSource.Webhook.Endpoint)),
		}
		if stripeEventSource.EventFilter != nil {
			params.EnabledEvents = stripe.StringSlice(stripeEventSource.EventFilter)
		}

		endpoint, err := webhookendpoint.New(params)
		if err != nil {
			return err
		}
		logger.With("endpoint-id", endpoint.ID).Info("new stripe webhook endpoint created")
	}
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (rc *Router) PostInactivate() error {
	return nil
}

func filterEvent(event *stripe.Event, filters []string) bool {
	if filters == nil {
		return true
	}
	for _, filter := range filters {
		if event.Type == filter {
			return true
		}
	}
	return false
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the Stripe event source...")
	defer sources.Recover(el.GetEventName())

	stripeEventSource := &el.StripeEventSource
	route := webhook.NewRoute(stripeEventSource.Webhook, log, el.GetEventSourceName(), el.GetEventName(), el.Metrics)

	return webhook.ManageRoute(ctx, &Router{
		route:             route,
		stripeEventSource: stripeEventSource,
	}, controller, dispatch)
}
