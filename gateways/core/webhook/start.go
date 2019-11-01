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
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/common/webhook"
	"github.com/ghodss/yaml"
)

// EventListener implements Eventing for webhook events
type EventListener struct {
	// Logger logs stuff
	Logger *logrus.Logger
}

// Router contains the configuration information for a route
type Router struct {
	// route contains information about a API endpoint
	route *webhook.Route
}

// controller controls the webhook operations
var (
	controller = webhook.NewController()
)

// set up the activation and inactivation channels to control the state of routes.
func init() {
	go webhook.ProcessRouteStatus(controller)
}

// webhook event payload
type payload struct {
	// Header is the http request header
	Header http.Header `json:"header"`
	// Body is http request body
	Body []byte `json:"body"`
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
	route := router.route

	logger := route.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: route.EventSource.Name,
			common.LabelEndpoint:    route.Context.Endpoint,
			common.LabelPort:        route.Context.Port,
			common.LabelHTTPMethod:  route.Context.Method,
		})

	logger.Info("a request received, processing it...")

	if !route.Active {
		logger.Info("endpoint is not active, wont't process the request")
		common.SendErrorResponse(writer, "endpoint is inactive")
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.WithError(err).Error("failed to parse request body")
		common.SendErrorResponse(writer, err.Error())
		return
	}

	data, err := json.Marshal(&payload{
		Header: request.Header,
		Body:   body,
	})
	if err != nil {
		logger.WithError(err).Error("failed to construct the event payload")
		common.SendErrorResponse(writer, err.Error())
		return
	}

	logger.Infoln("dispatching event on route's data channel...")
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

// StartEventSource starts a event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	log := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	log.Info("started operating on the event source...")

	var webhookEventSource *webhook.Context
	if err := yaml.Unmarshal(eventSource.Value, &webhookEventSource); err != nil {
		log.WithError(err).Error("failed to parse the event source")
		return err
	}

	route := webhook.NewRoute(webhookEventSource, listener.Logger, eventSource)

	return webhook.ManageRoute(&Router{
		route: route,
	}, controller, eventStream)
}
