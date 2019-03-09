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

package aws_sns

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
)

const (
	LabelSNSConfig       = "snsConfig"
	LabelSNSSession      = "snsSession"
	LabelSubscriptionArn = "subscriptionArn"
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
		common.SendErrorResponse(writer, fmt.Sprintf("failed to parse request. err: %+v", err))
		return
	}

	var snspayload *httpNotification
	err = yaml.Unmarshal(body, &snspayload)
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert request payload into snsConfig payload")
		return
	}

	sc := rc.Configs[LabelSNSConfig].(*snsConfig)

	switch snspayload.Type {
	case MESSAGE_TYPE_SUBSCRIPTION_CONFIRMATION:
		awsSession := rc.Configs[LabelSNSSession].(*snslib.SNS)
		out, err := awsSession.ConfirmSubscription(&snslib.ConfirmSubscriptionInput{
			TopicArn: &sc.TopicArn,
			Token:    &snspayload.Token,
		})
		if err != nil {
			logger.Error().Err(err).Msg("failed to send confirmation response to amazon")
			common.SendErrorResponse(writer, "failed to confirm subscription")
			return
		}
		rc.Configs[LabelSubscriptionArn] = out.SubscriptionArn

	case MESSAGE_TYPE_NOTIFICATION:
		helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh <- body
	}

	response = "request successfully processed"
	logger.Info().Msg(response)
	common.SendSuccessResponse(writer, response)
}

func (ese *SNSEventSourceExecutor) PostActivate(rc *gwcommon.RouteConfig) error {
	logger := rc.Log.With().Str("event-source", rc.EventSource.Name).Str("endpoint", rc.Webhook.Endpoint).
		Str("port", rc.Webhook.Port).Logger()

	sc := rc.Configs[LabelSNSConfig].(*snsConfig)

	creds, err := gwcommon.GetAWSCreds(ese.Clientset, ese.Namespace, sc.AccessKey, sc.SecretKey)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get aws credentials")
	}

	awsSession, err := gwcommon.GetAWSSession(creds, sc.Region)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create new session")
		return err
	}

	logger.Info().Msg("subscribing to sns topic")

	snsSession := snslib.New(awsSession)
	rc.Configs[LabelSNSSession] = snsSession
	formattedUrl := gwcommon.GenerateFormattedURL(sc.Hook)

	if _, err := snsSession.Subscribe(&snslib.SubscribeInput{
		Endpoint: &formattedUrl,
		Protocol: &snsProtocol,
		TopicArn: &sc.TopicArn,
	}); err != nil {
		logger.Error().Err(err).Msg("failed to send subscribe request")
		return err
	}

	return nil
}

// PostStop unsubscribes the topic
func PostStop(rc *gwcommon.RouteConfig) error {
	awsSession := rc.Configs[LabelSNSSession].(*snslib.SNS)
	if _, err := awsSession.Unsubscribe(&snslib.UnsubscribeInput{
		SubscriptionArn: rc.Configs[LabelSubscriptionArn].(*string),
	}); err != nil {
		rc.Log.Error().Err(err).Str("event-source-name", rc.EventSource.Name).Msg("failed to unsubscribe")
		return err
	}
	return nil
}

// StartConfig runs a configuration
func (ese *SNSEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		ese.Log.Error().Err(err).Str("event-source-name", eventSource.Name).Msg("failed to parse event source")
		return err
	}
	sc := config.(*snsConfig)

	return gwcommon.ProcessRoute(&gwcommon.RouteConfig{
		Webhook: sc.Hook,
		Configs: map[string]interface{}{
			LabelSNSConfig: sc,
		},
		Log:                ese.Log,
		EventSource:        eventSource,
		PostActivate:       ese.PostActivate,
		PostStop:           PostStop,
		RouteActiveHandler: RouteActiveHandler,
		StartCh:            make(chan struct{}),
	}, helper, eventStream)
}
