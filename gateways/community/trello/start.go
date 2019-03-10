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

package trello

import (
	"fmt"
	"io/ioutil"
	"net/http"

	trellolib "github.com/adlio/trello"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/store"
)

const (
	LabelTrelloConfig = "trelloConfig"
	LabelTrelloClient = "trelloClient"
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

	logger := rc.Log.With().Str("event-source", rc.EventSource.Name).Str("endpoint", rc.Webhook.Endpoint).
		Str("port", rc.Webhook.Port).
		Str("http-method", request.Method).Logger()
	logger.Info().Msg("request received")

	if !helper.ActiveEndpoints[rc.Webhook.Endpoint].Active {
		response = fmt.Sprintf("the route: endpoint %s and method %s is deactived", rc.Webhook.Endpoint, rc.Webhook.Method)
		logger.Info().Msg("endpoint is not active")
		common.SendErrorResponse(writer, response)
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse request body")
		return
	}

	helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh <- body

	response = "request successfully processed"
	logger.Info().Msg(response)
	common.SendSuccessResponse(writer, response)
}

func (ese *TrelloEventSourceExecutor) PostActivate(rc *gwcommon.RouteConfig) error {
	logger := rc.Log.With().Str("event-source", rc.EventSource.Name).Str("endpoint", rc.Webhook.Endpoint).
		Str("port", rc.Webhook.Port).Logger()

	tl := rc.Configs[LabelTrelloConfig].(*trello)

	logger.Info().Msg("retrieving apikey and token")
	apiKey, err := store.GetSecrets(ese.Clientset, ese.Namespace, tl.ApiKey.Name, tl.ApiKey.Key)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve api key")
		return err
	}
	token, err := store.GetSecrets(ese.Clientset, ese.Namespace, tl.Token.Name, tl.Token.Key)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve token")
		return err
	}

	client := trellolib.NewClient(apiKey, token)

	formattedUrl := gwcommon.GenerateFormattedURL(tl.Hook)

	if err = client.CreateWebhook(&trellolib.Webhook{
		Active:      true,
		Description: tl.Description,
		CallbackURL: formattedUrl,
	}); err != nil {
		logger.Error().Err(err).Msg("failed to create webhook")
		return err
	}

	rc.Configs[LabelTrelloClient] = client

	logger.Info().Msg("webhook created")
	return nil
}

// StartConfig runs a configuration
func (ese *TrelloEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		ese.Log.Error().Err(err).Str("event-source-name", eventSource.Name).Msg("failed to parse event source")
		return err
	}
	tl := config.(*trello)

	return gwcommon.ProcessRoute(&gwcommon.RouteConfig{
		Webhook: tl.Hook,
		Configs: map[string]interface{}{
			LabelTrelloConfig: tl,
		},
		Log:                ese.Log,
		EventSource:        eventSource,
		PostActivate:       ese.PostActivate,
		PostStop:           gwcommon.DefaultPostStop,
		RouteActiveHandler: RouteActiveHandler,
		StartCh:            make(chan struct{}),
	}, helper, eventStream)
}
