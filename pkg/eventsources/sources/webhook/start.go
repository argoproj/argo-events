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

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
	"go.uber.org/zap"
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

	webhook.DispatchEvent(route, data, logger, writer)
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
