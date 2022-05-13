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

package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
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
	WebhookContext  v1alpha1.WebhookContext
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
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.WebhookEvent
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
		common.SendErrorResponse(writer, "endpoint is inactive")
		return
	}

	defer func(start time.Time) {
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	request.Body = http.MaxBytesReader(writer, request.Body, 65536)
	body, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Errorw("failed to parse request body", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	payload := &events.WebhookEventData{
		Header:   request.Header,
		Body:     (*json.RawMessage)(&body),
		Metadata: route.Context.Metadata,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		logger.Errorw("failed to construct the event payload", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	logger.Info("dispatching event on route's data channel...")
	route.DataCh <- data
	logger.Info("successfully processed the request")
	common.SendSuccessResponse(writer, "success")
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

	route := webhook.NewRoute(&el.WebhookContext, log, el.GetEventSourceName(), el.GetEventName(), el.Metrics)
	return webhook.ManageRoute(ctx, &Router{
		route: route,
	}, controller, dispatch)
}
