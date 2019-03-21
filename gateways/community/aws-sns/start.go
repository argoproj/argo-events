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

var (
	helper = gwcommon.NewWebhookHelper()
)

func init() {
	go gwcommon.InitRouteChannels(helper)
}

// GetRoute returns the route
func (rc *RouteConfig) GetRoute() *gwcommon.Route {
	return rc.Route
}

// RouteHandler handles new routes
func (rc *RouteConfig) RouteHandler(writer http.ResponseWriter, request *http.Request) {
	r := rc.Route

	logger := r.Logger.With().
		Str(common.LabelEventSource, r.EventSource.Name).
		Str("endpoint", r.Webhook.Endpoint).
		Str("port", r.Webhook.Port).
		Str("http-method", request.Method).Logger()

	logger.Info().Msg("request received")

	if !helper.ActiveEndpoints[r.Webhook.Endpoint].Active {
		logger.Info().Msg("endpoint is not active")
		common.SendErrorResponse(writer, "")
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse request body")
		common.SendErrorResponse(writer, "")
		return
	}

	var snspayload *httpNotification
	err = yaml.Unmarshal(body, &snspayload)
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert request payload into sns event source payload")
		common.SendErrorResponse(writer, "")
		return
	}

	switch snspayload.Type {
	case messageTypeSubscriptionConfirmation:
		awsSession := rc.session
		out, err := awsSession.ConfirmSubscription(&snslib.ConfirmSubscriptionInput{
			TopicArn: &rc.snses.TopicArn,
			Token:    &snspayload.Token,
		})
		if err != nil {
			logger.Error().Err(err).Msg("failed to send confirmation response to amazon")
			common.SendErrorResponse(writer, "")
			return
		}
		rc.subscriptionArn = out.SubscriptionArn

	case messageTypeNotification:
		helper.ActiveEndpoints[r.Webhook.Endpoint].DataCh <- body
	}

	logger.Info().Msg("request successfully processed")
}

// PostStart subscribes to the sns topic
func (rc *RouteConfig) PostStart() error {
	r := rc.Route

	logger := r.Logger.With().
		Str(common.LabelEventSource, r.EventSource.Name).
		Str("endpoint", r.Webhook.Endpoint).
		Str("port", r.Webhook.Port).
		Str("topic-arn", rc.snses.TopicArn).
		Logger()
	logger.Info().Msg("subscribing to sns topic")

	sc := rc.snses
	creds, err := gwcommon.GetAWSCreds(rc.clientset, rc.namespace, sc.AccessKey, sc.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to get aws credentials. err: %+v", err)
	}

	awsSession, err := gwcommon.GetAWSSession(creds, sc.Region)
	if err != nil {
		return fmt.Errorf("failed to create aws session. err: %+v", err)
	}

	rc.session = snslib.New(awsSession)
	formattedUrl := gwcommon.GenerateFormattedURL(sc.Hook)
	if _, err := rc.session.Subscribe(&snslib.SubscribeInput{
		Endpoint: &formattedUrl,
		Protocol: &snsProtocol,
		TopicArn: &sc.TopicArn,
	}); err != nil {
		return fmt.Errorf("failed to send subscribe request. err: %+v", err)
	}

	return nil
}

// PostStop unsubscribes from the sns topic
func (rc *RouteConfig) PostStop() error {
	if _, err := rc.session.Unsubscribe(&snslib.UnsubscribeInput{
		SubscriptionArn: rc.subscriptionArn,
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe. err: %+v", err)
	}
	return nil
}

// StartEventSource starts an SNS event source
func (ese *SNSEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str(common.LabelEventSource, eventSource.Name).Msg("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		ese.Log.Error().Err(err).Str(common.LabelEventSource, eventSource.Name).Msg("failed to parse event source")
		return err
	}
	sc := config.(*snsEventSource)

	return gwcommon.ProcessRoute(&RouteConfig{
		Route: &gwcommon.Route{
			Logger:      &ese.Log,
			EventSource: eventSource,
			StartCh:     make(chan struct{}),
			Webhook:     sc.Hook,
		},
		snses:     sc,
		namespace: ese.Namespace,
		clientset: ese.Clientset,
	}, helper, eventStream)
}
