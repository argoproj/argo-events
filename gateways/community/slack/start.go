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
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/store"
	"github.com/nlopes/slack/slackevents"
)

var (
	helper = gwcommon.NewWebhookHelper()
)

func init() {
	go gwcommon.InitRouteChannels(helper)
}

func (rc *RouteConfig) GetRoute() *gwcommon.Route {
	return rc.route
}

// RouteHandler handles new route
func (rc *RouteConfig) RouteHandler(writer http.ResponseWriter, request *http.Request) {
	r := rc.route

	logger := r.Logger.With().
		Str("event-source", r.EventSource.Name).
		Str("endpoint", r.Webhook.Endpoint).
		Str("port", r.Webhook.Port).
		Str("http-method", request.Method).
		Logger()

	logger.Info().Msg("request received")

	if !helper.ActiveEndpoints[r.Webhook.Endpoint].Active {
		logger.Warn().Msg("endpoint is not active")
		common.SendErrorResponse(writer, "")
		return
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(request.Body); err != nil {
		logger.Error().Err(err).Msg("failed to parse request body")
		common.SendInternalErrorResponse(writer, "")
		return
	}

	body := buf.String()
	eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: rc.token}))
	if e != nil {
		logger.Error().Msg("failed to extract event")
		common.SendInternalErrorResponse(writer, "")
		return
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			logger.Error().Msg("failed to verify the challenge")
			common.SendInternalErrorResponse(writer, "")
			return
		}
		writer.Header().Set("Content-Type", "text")
		if _, err := writer.Write([]byte(r.Challenge)); err != nil {
			logger.Error().Err(err).Msg("failed to write the response for url verification")
			// don't return, we want to keep this running to give user chance to retry
		}
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		data, err := json.Marshal(eventsAPIEvent.InnerEvent.Data)
		if err != nil {
			logger.Error().Err(err).Msg("failed to marshal event data")
			common.SendInternalErrorResponse(writer, "")
			return
		}
		helper.ActiveEndpoints[rc.route.Webhook.Endpoint].DataCh <- data
	}

	logger.Info().Msg("request successfully processed")
	common.SendSuccessResponse(writer, "")
}

func (rc *RouteConfig) PostStart() error {
	return nil
}

func (rc *RouteConfig) PostStop() error {
	return nil
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

	ses := config.(*slackEventSource)

	token, err := store.GetSecrets(ese.Clientset, ese.Namespace, ses.Token.Name, ses.Token.Key)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve token")
		return err
	}

	return gwcommon.ProcessRoute(&RouteConfig{
		route: &gwcommon.Route{
			Logger:      &ese.Log,
			StartCh:     make(chan struct{}),
			Webhook:     ses.Hook,
			EventSource: eventSource,
		},
		token:     token,
		clientset: ese.Clientset,
		namespace: ese.Namespace,
		ses:       ses,
	}, helper, eventStream)
}
