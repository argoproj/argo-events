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

package stripe

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhookendpoint"
)

// controller controls the webhook operations
var (
	controller = webhook.NewController()
)

// set up the activation and inactivation channels to control the state of routes.
func init() {
	go webhook.ProcessRouteStatus(controller)
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

	logger := route.Logger.WithFields(
		logrus.Fields{
			common.LabelEventSource: route.EventSource.Name,
			common.LabelEndpoint:    route.Context.Endpoint,
			common.LabelHTTPMethod:  route.Context.Method,
		})

	logger.Info("request a received, processing it...")

	if !route.Active {
		logger.Warn("endpoint is not active, won't process it")
		common.SendErrorResponse(writer, "endpoint is inactive")
		return
	}

	const MaxBodyBytes = int64(65536)
	request.Body = http.MaxBytesReader(writer, request.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.WithError(err).Errorln("error reading request body")
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event := stripe.Event{}

	if err := json.Unmarshal(payload, &event); err != nil {
		logger.WithError(err).Errorln("failed to parse webhook body json")
		common.SendErrorResponse(writer, "failed to parse the event")
		return
	}

	ok := filterEvent(event, rc.stripeEventSource.EventFilter)
	if !ok {
		logger.WithField("event-type", event.Type).Warnln("failed to pass the filters")
		common.SendSuccessResponse(writer, "success")
		return
	}

	response := Response{
		Id:        event.ID,
		EventType: event.Type,
		Data:      event.Data.Raw,
	}

	data, err := json.Marshal(response)
	if err != nil {
		logger.WithField("event-id", event.ID).Warnln("failed to marshal event into gateway response")
		common.SendSuccessResponse(writer, "success")
		return
	}

	logger.Infoln("dispatching event on route's data channel...")
	route.DataCh <- data
	logger.Info("request successfully processed")
	common.SendSuccessResponse(writer, "success")
}

// PostActivate performs operations once the route is activated and ready to consume requests
func (rc *Router) PostActivate() error {
	if rc.stripeEventSource.CreateWebhook {
		route := rc.route
		stripeEventSource := rc.stripeEventSource
		logger := route.Logger.WithFields(
			logrus.Fields{
				common.LabelEventSource: route.EventSource.Name,
				common.LabelEndpoint:    route.Context.Endpoint,
				common.LabelHTTPMethod:  route.Context.Method,
			})
		logger.Infoln("registering a new webhook")

		apiKey, err := common.GetSecrets(rc.k8sClient, stripeEventSource.Namespace, stripeEventSource.APIKey)
		if err != nil {
			return err
		}

		stripe.Key = apiKey

		params := &stripe.WebhookEndpointParams{
			URL: stripe.String(common.FormattedURL(stripeEventSource.Webhook.URL, stripeEventSource.Webhook.Endpoint)),
		}
		if stripeEventSource.EventFilter != nil {
			params.EnabledEvents = stripe.StringSlice(stripeEventSource.EventFilter)
		}

		endpoint, err := webhookendpoint.New(params)
		if err != nil {
			return err
		}

		logger.WithField("endpoint-id", endpoint.ID).Infoln("new stripe webhook endpoint created")
	}
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (rc *Router) PostInactivate() error {
	return nil
}

func filterEvent(event stripe.Event, filters []string) bool {
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

// StartEventSource starts a event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer server.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("started processing the event source...")

	logger.Infoln("parsing stripe event source...")

	var stripeEventSource *v1alpha1.StripeEventSource
	if err := yaml.Unmarshal(eventSource.Value, &stripeEventSource); err != nil {
		logger.WithError(err).Errorln("failed to parse the event source")
		return err
	}

	route := webhook.NewRoute(stripeEventSource.Webhook, listener.Logger, eventSource)

	return webhook.ManageRoute(&Router{
		route:             route,
		k8sClient:         listener.K8sClient,
		stripeEventSource: stripeEventSource,
	}, controller, eventStream)
}
