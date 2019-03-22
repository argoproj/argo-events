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
	"fmt"
	"github.com/argoproj/argo-events/common"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/gateways"
)

var (
	helper = gwcommon.NewWebhookHelper()
)

func init() {
	go gwcommon.InitRouteChannels(helper)
}

func (rc *RouteConfig) GetRoute() *gwcommon.Route {
	return rc.Route
}

// RouteHandler handles new route
func (rc *RouteConfig) RouteHandler(writer http.ResponseWriter, request *http.Request) {
	var response string

	r := rc.Route

	logger := r.Logger.WithEventSource(r.EventSource.Name).WithEndpoint(r.Webhook.Endpoint).WithPort(r.Webhook.Port).WithHttpMethod(request.Method)

	logger.Info().Msg("request received")

	if !helper.ActiveEndpoints[r.Webhook.Endpoint].Active {
		response = fmt.Sprintf("the route: endpoint %s and method %s is deactived", r.Webhook.Endpoint, r.Webhook.Method)
		logger.Info().Msg("endpoint is not active")
		common.SendErrorResponse(writer, response)
		return
	}

	if r.Webhook.Method != request.Method {
		logger.Warn().Str("expected", r.Webhook.Method).Str("actual", request.Method).Msg("method mismatch")
		common.SendErrorResponse(writer, fmt.Sprintf("the method %s is not defined for endpoint %s", r.Webhook.Method, r.Webhook.Endpoint))
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse request body")
		common.SendErrorResponse(writer, fmt.Sprintf("failed to parse request. err: %+v", err))
		return
	}

	helper.ActiveEndpoints[r.Webhook.Endpoint].DataCh <- body
	response = "request successfully processed"
	logger.Info().Msg(response)
	common.SendSuccessResponse(writer, response)
}

func (rc *RouteConfig) PostStart() error {
	return nil
}

func (rc *RouteConfig) PostStop() error {
	return nil
}

// StartEventSource starts a event source
func (ese *WebhookEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.WithEventSource(eventSource.Name).Info().Msg("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		ese.Log.WithEventSource(eventSource.Name).Info().Msg("failed to parse event source")
		return err
	}
	h := config.(*gwcommon.Webhook)
	h.Endpoint = gwcommon.FormatWebhookEndpoint(h.Endpoint)

	return gwcommon.ProcessRoute(&RouteConfig{
		Route: &gwcommon.Route{
			Logger:      ese.Log,
			EventSource: eventSource,
			StartCh:     make(chan struct{}),
			Webhook:     h,
		},
	}, helper, eventStream)
}
