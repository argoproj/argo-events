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

// routeActiveHandler handles new route
func RouteActiveHandler(writer http.ResponseWriter, request *http.Request, rc *gwcommon.RouteConfig) {
	var response string
	if !helper.ActiveEndpoints[rc.Webhook.Endpoint].Active {
		response = fmt.Sprintf("the route: endpoint %s and method %s is deactived", rc.Webhook.Endpoint, rc.Webhook.Method)
		rc.Log.Info().Str("endpoint", rc.Webhook.Endpoint).Str("http-method", request.Method).Str("response", response).Msg("endpoint is not active")
		common.SendErrorResponse(writer, response)
		return
	}
	if rc.Webhook.Method != request.Method {
		msg := fmt.Sprintf("the method %s is not defined for endpoint %s", rc.Webhook.Method, rc.Webhook.Endpoint)
		rc.Log.Info().Str("endpoint", rc.Webhook.Endpoint).Str("http-method", request.Method).Str("response", response).Msg("endpoint is not active")
		common.SendErrorResponse(writer, msg)
		return
	}

	rc.Log.Info().Str("endpoint", rc.Webhook.Endpoint).Str("http-method", request.Method).Msg("payload received")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		rc.Log.Error().Err(err).Str("endpoint", rc.Webhook.Endpoint).Str("http-method", request.Method).Str("response", response).Msg("failed to parse request body")
		common.SendErrorResponse(writer, fmt.Sprintf("failed to parse request. err: %+v", err))
		return
	}

	helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh <- body
	response = "request successfully processed"
	rc.Log.Info().Str("endpoint", rc.Webhook.Endpoint).Str("http-method", request.Method).Str("response", response).Msg("request payload parsed successfully")
	common.SendSuccessResponse(writer, response)
}

// StartEventSource starts a event source
func (ese *WebhookEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	h, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	return gwcommon.ProcessRoute(&gwcommon.RouteConfig{
		Webhook:     h,
		Log:         ese.Log,
		EventSource: eventSource,
		PostActivate: func() error {
			return nil
		},
		RouteActiveHandler: RouteActiveHandler,
		StartCh:            make(chan struct{}),
	}, helper, eventStream)
}
