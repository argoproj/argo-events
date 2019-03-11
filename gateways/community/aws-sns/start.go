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
	"github.com/argoproj/argo-events/store"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
)

const (
	labelSNSConfig       = "snsConfig"
	labelSNSSession      = "snsSession"
	labelSubscriptionArn = "subscriptionArn"
)

var (
	helper = gwcommon.NewWebhookHelper()
)

func init() {
	go gwcommon.InitRouteChannels(helper)
}

// RouteActiveHandler handles new routes
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

	sc := rc.Configs[labelSNSConfig].(*snsConfig)

	switch snspayload.Type {
	case messageTypeSubscriptionConfirmation:
		awsSession := rc.Configs[labelSNSSession].(*snslib.SNS)
		out, err := awsSession.ConfirmSubscription(&snslib.ConfirmSubscriptionInput{
			TopicArn: &sc.TopicArn,
			Token:    &snspayload.Token,
		})
		if err != nil {
			logger.Error().Err(err).Msg("failed to send confirmation response to amazon")
			return
		}
		rc.Configs[labelSubscriptionArn] = out.SubscriptionArn

	case messageTypeNotification:
		helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh <- body
	}

	response = "request successfully processed"
	logger.Info().Msg(response)
	common.SendSuccessResponse(writer, response)
}

// PostActivate subscribes to the sns topic
func (ese *SNSEventSourceExecutor) PostActivate(rc *gwcommon.RouteConfig) error {
	logger := rc.Log.With().Str("event-source", rc.EventSource.Name).Str("endpoint", rc.Webhook.Endpoint).
		Str("port", rc.Webhook.Port).Logger()

	sc := rc.Configs[labelSNSConfig].(*snsConfig)
	// retrieve access key id and secret access key
	accessKey, err := store.GetSecrets(ese.Clientset, ese.Namespace, sc.AccessKey.Name, sc.AccessKey.Key)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve access key")
		return err
	}
	secretKey, err := store.GetSecrets(ese.Clientset, ese.Namespace, sc.SecretKey.Name, sc.SecretKey.Key)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve secret key")
		return err
	}

	creds := credentials.NewStaticCredentialsFromCreds(credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})

	formattedUrl := gwcommon.GenerateFormattedURL(sc.Hook)

	awsSession, err := session.NewSession(&aws.Config{
		Region:      &sc.Region,
		Credentials: creds,
		HTTPClient:  &http.Client{},
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to create new session")
		return err
	}

	snsSession := snslib.New(awsSession)
	rc.Configs[labelSNSSession] = snsSession

	logger.Info().Msg("subscribing to snsConfig topic")
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

// PostStop unsubscribes from the sns topic
func PostStop(rc *gwcommon.RouteConfig) error {
	awsSession := rc.Configs[labelSNSSession].(*snslib.SNS)
	if _, err := awsSession.Unsubscribe(&snslib.UnsubscribeInput{
		SubscriptionArn: rc.Configs[labelSubscriptionArn].(*string),
	}); err != nil {
		rc.Log.Error().Err(err).Str("event-source-name", rc.EventSource.Name).Msg("failed to unsubscribe")
		return err
	}
	return nil
}

// StartEventSource starts an SNS event source
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
			labelSNSConfig: sc,
		},
		Log:                ese.Log,
		EventSource:        eventSource,
		PostActivate:       ese.PostActivate,
		PostStop:           PostStop,
		RouteActiveHandler: RouteActiveHandler,
		StartCh:            make(chan struct{}),
	}, helper, eventStream)
}
