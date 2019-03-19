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

package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/store"
	"github.com/nlopes/slack/slackevents"
)

const (
	labelSlackToken = "slackToken"
)

var (
	helper = gwcommon.NewWebhookHelper()
)

func init() {
	go gwcommon.InitRouteChannels(helper)
}

// RouteHandler handles new route
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

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(request.Body); err != nil {
		logger.Error().Err(err).Msg("failed to parse request body")
		common.SendInternalErrorResponse(writer, fmt.Sprintf("failed to parse request. err: %+v", err))
		return
	}

	body := buf.String()
	token := rc.Configs[labelSlackToken]
	eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: token.(string)}))
	if e != nil {
		response = "failed to extract event"
		logger.Error().Msg(response)
		common.SendInternalErrorResponse(writer, "failed to extract event")
		return
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			response = "failed to verify the challenge"
			logger.Error().Msg(response)
			common.SendInternalErrorResponse(writer, response)
			return
		}
		writer.Header().Set("Content-Type", "text")
		writer.Write([]byte(r.Challenge))
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		data, err := json.Marshal(eventsAPIEvent.InnerEvent.Data)
		if err != nil {
			response = "failed to marshal event data"
			logger.Error().Err(err).Msg(response)
			common.SendInternalErrorResponse(writer, response)
			return
		}
		helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh <- data
	}

	response = "request successfully processed"
	logger.Info().Msg(response)
	common.SendSuccessResponse(writer, response)
}

// StartEventSource starts a event source
func (ese *SlackEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	logger := ese.Log.With().Str("event-source-name", eventSource.Name).Logger()
	logger.Info().Msg("operating on event source")

	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse event source")
		return err
	}
	sc := config.(*slackConfig)

	token, err := store.GetSecrets(ese.Clientset, ese.Namespace, sc.Token.Name, sc.Token.Key)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve token")
		return err
	}

	return gwcommon.ProcessRoute(&gwcommon.RouteConfig{
		Webhook: sc.Hook,
		Configs: map[string]interface{}{
			labelSlackToken: token,
		},
		Log:                ese.Log,
		EventSource:        eventSource,
		PostActivate:       gwcommon.DefaultPostActivate,
		PostStop:           gwcommon.DefaultPostStop,
		RouteActiveHandler: RouteActiveHandler,
		StartCh:            make(chan struct{}),
	}, helper, eventStream)
}
