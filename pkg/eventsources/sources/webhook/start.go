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

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	"github.com/argoproj/argo-events/pkg/shared/tracing"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

var (
	controller = webhook.NewController()
)

func init() {
	go webhook.ProcessRouteStatus(controller)
}

// EventListener implements Eventing for webhook events
type EventListener struct {
	EventSourceName string
	EventName       string
	Webhook         v1alpha1.WebhookEventSource
	Metrics         *metrics.Metrics
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
	return v1alpha1.WebhookEvent
}

// Router contains the configuration information for a route
type Router struct {
	// route contains information about a API endpoint
	route *webhook.Route
}

// Implement Router
// 1. GetRoute
// 2. HandleRoute
// 3. PostActivate
// 4. PostDeactivate

// GetRoute returns the route
func (router *Router) GetRoute() *webhook.Route {
	return router.route
}

// HandleRoute handles incoming requests on the route
func (router *Router) HandleRoute(writer http.ResponseWriter, request *http.Request) {
	route := router.GetRoute()

	// Extract any upstream trace context from the request headers and start a
	// SERVER span for this inbound webhook. This creates the gateway ->
	// eventsource edge in the distributed trace and becomes the parent of the
	// eventsource.publish PRODUCER span.
	if request.Header == nil {
		request.Header = make(http.Header)
	}
	httpRoute := ""
	if request.URL != nil {
		httpRoute = request.URL.Path
	}
	ctx := otel.GetTextMapPropagator().Extract(request.Context(), propagation.HeaderCarrier(request.Header))
	ctx, receiveSpan := tracing.StartServerSpan(ctx, otel.Tracer("argo-events-eventsource"), "eventsource.receive",
		attribute.String("eventsource.name", route.EventSourceName),
		attribute.String("eventsource.type", "webhook"),
		attribute.String("http.method", request.Method),
		attribute.String("http.route", httpRoute),
	)
	defer receiveSpan.End()
	// Inject the SERVER span's trace context into the request headers so that
	// WithHTTPHeaders (applied below) propagates it into the outgoing CloudEvent
	// extensions, making eventsource.publish a child of eventsource.receive.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(request.Header))
	request = request.WithContext(ctx)

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
	)

	logger.Info("a request received, processing it...")

	if !route.Active {
		logger.Info("endpoint is not active, wont't process the request")
		sharedutil.SendErrorResponse(writer, "endpoint is inactive")
		return
	}

	if route.Context.Method != request.Method {
		logger.Info("http method does not match")
		sharedutil.SendErrorResponse(writer, "http method does not match")
		return
	}

	defer func(start time.Time) {
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	body, err := GetBody(&writer, request, route, logger)
	if err != nil {
		logger.Errorw("failed to get body", zap.Error(err))
		sharedutil.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	payload := &events.WebhookEventData{
		Header:   request.Header,
		Body:     body,
		Metadata: route.Context.Metadata,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		logger.Errorw("failed to construct the event payload", zap.Error(err))
		sharedutil.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	// Detect if the incoming request is a CloudEvent and preserve its metadata.
	// The request body was already consumed by GetBody, so we reconstruct it
	// for the CloudEvents SDK parser.
	var opts []eventsourcecommon.Option
	if incomingCE := parseIncomingCloudEvent(request, body); incomingCE != nil {
		logger.Info("incoming CloudEvent detected, preserving metadata")
		opts = append(opts, eventsourcecommon.WithCloudEvent(*incomingCE))
	}
	// Always propagate the SERVER span's trace context (injected above into
	// request.Header) into the outgoing CloudEvent extensions. This ensures
	// eventsource.publish becomes a child of eventsource.receive regardless of
	// whether the original request was a CloudEvent or a plain HTTP request.
	// When applied after WithCloudEvent, this overwrites any incoming
	// traceparent with the SERVER span so the publish span is parented correctly.
	opts = append(opts, eventsourcecommon.WithHTTPHeaders(request.Header))

	webhook.DispatchEvent(route, data, logger, writer, opts...)
}

// PostActivate performs operations once the route is activated and ready to consume requests
func (router *Router) PostActivate() error {
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (router *Router) PostInactivate() error {
	return nil
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the webhook event source...")

	route := webhook.NewRoute(&el.Webhook.WebhookContext, log, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	return webhook.ManageRoute(ctx, &Router{
		route: route,
	}, controller, dispatch)
}

func GetBody(writer *http.ResponseWriter, request *http.Request, route *webhook.Route, logger *zap.SugaredLogger) (*json.RawMessage, error) {
	switch request.Method {
	case http.MethodGet:
		body, _ := json.Marshal(request.URL.Query())
		ret := json.RawMessage(body)
		return &ret, nil
	case http.MethodPost:
		contentType := ""
		if len(request.Header["Content-Type"]) > 0 {
			contentType = request.Header["Content-Type"][0]
		}

		switch contentType {
		case "application/x-www-form-urlencoded":
			if err := request.ParseForm(); err != nil {
				logger.Errorw("failed to parse form data", zap.Error(err))
				sharedutil.SendInternalErrorResponse(*writer, err.Error())
				route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
				return nil, err
			}
			body, _ := json.Marshal(request.PostForm)
			ret := json.RawMessage(body)
			return &ret, nil
		// default including "application/json" is parsing body as JSON
		default:
			request.Body = http.MaxBytesReader(*writer, request.Body, route.Context.GetMaxPayloadSize())
			body, err := getRequestBody(request)
			if err != nil {
				logger.Errorw("failed to read request body", zap.Error(err))
				sharedutil.SendErrorResponse(*writer, err.Error())
				route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
				return nil, err
			}
			ret := json.RawMessage(body)
			return &ret, nil
		}
	default:
		return nil, fmt.Errorf("unsupoorted method: %s", request.Method)
	}
}

func getRequestBody(request *http.Request) ([]byte, error) {
	// Read request payload
	body, err := io.ReadAll(request.Body)
	// Reset request.Body ReadCloser to prevent side-effect if re-read
	request.Body = io.NopCloser(bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse request body, %w", err)
	}
	return body, nil
}

// parseIncomingCloudEvent attempts to parse the HTTP request as a CloudEvent.
// The original request body has already been consumed, so we reconstruct it
// from the body parameter. Returns nil if the request is not a CloudEvent.
func parseIncomingCloudEvent(request *http.Request, body *json.RawMessage) *cloudevents.Event {
	if body == nil {
		return nil
	}
	// Reconstruct an http.Request with the body restored for the CE SDK parser.
	reqCopy := request.Clone(request.Context())
	reqCopy.Body = io.NopCloser(bytes.NewReader(*body))
	ce, err := cehttp.NewEventFromHTTPRequest(reqCopy)
	if err != nil {
		// Not a CloudEvent — this is the normal case for plain HTTP requests.
		return nil
	}
	return ce
}
