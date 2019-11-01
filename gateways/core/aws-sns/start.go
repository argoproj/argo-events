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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/gateways/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/aws/aws-sdk-go/aws/session"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"net/http"
)

var (
	// controller controls the webhook operations
	controller = webhook.NewController()
)

// set up route activation and deactivation channels
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
	return rc.Route
}

// HandleRoute handles new routes
func (rc *Router) HandleRoute(writer http.ResponseWriter, request *http.Request) {
	r := rc.Route

	logger := r.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: r.EventSource.Name,
			common.LabelEndpoint:    r.Webhook.Endpoint,
			common.LabelPort:        r.Webhook.Port,
			common.LabelHTTPMethod:  r.Webhook.Method,
		})

	logger.Info("request received")

	if !controller.ActiveEndpoints[r.Webhook.Endpoint].Active {
		logger.Info("endpoint is not active")
		common.SendErrorResponse(writer, "")
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.WithError(err).Error("failed to parse request body")
		common.SendErrorResponse(writer, "")
		return
	}

	var snspayload *httpNotification
	err = yaml.Unmarshal(body, &snspayload)
	if err != nil {
		logger.WithError(err).Error("failed to convert request payload into sns event source payload")
		common.SendErrorResponse(writer, "")
		return
	}

	switch snspayload.Type {
	case messageTypeSubscriptionConfirmation:
		awsSession := rc.session
		out, err := awsSession.ConfirmSubscription(&snslib.ConfirmSubscriptionInput{
			TopicArn: &rc.eventSource.TopicArn,
			Token:    &snspayload.Token,
		})
		if err != nil {
			logger.WithError(err).Error("failed to send confirmation response to amazon")
			common.SendErrorResponse(writer, "")
			return
		}
		rc.subscriptionArn = out.SubscriptionArn

	case messageTypeNotification:
		controller.ActiveEndpoints[r.Webhook.Endpoint].DataCh <- body
	}

	logger.Info("request successfully processed")
}

// PostStart subscribes to the sns topic
func (rc *Router) PostStart() error {
	r := rc.Route

	logger := r.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: r.EventSource.Name,
			common.LabelEndpoint:    r.Webhook.Endpoint,
			common.LabelPort:        r.Webhook.Port,
			common.LabelHTTPMethod:  r.Webhook.Method,
			"topic-arn":             rc.eventSource.TopicArn,
		})

	logger.Info("subscribing to sns topic")

	sc := rc.eventSource
	var awsSession *session.Session

	if sc.AccessKey == nil && sc.SecretKey == nil {
		awsSessionWithoutCreds, err := gwcommon.GetAWSSessionWithoutCreds(sc.Region)
		if err != nil {
			return fmt.Errorf("failed to create aws session. err: %+v", err)
		}

		awsSession = awsSessionWithoutCreds
	} else {
		creds, err := gwcommon.GetAWSCreds(rc.k8sClient, rc.namespace, sc.AccessKey, sc.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to create aws session. err: %+v", err)
		}

		awsSessionWithCreds, err := gwcommon.GetAWSSession(creds, sc.Region)
		if err != nil {
			return fmt.Errorf("failed to create aws session. err: %+v", err)
		}

		awsSession = awsSessionWithCreds
	}

	rc.session = snslib.New(awsSession)
	formattedUrl := gwcommon.GenerateFormattedURL(sc.WebHook)
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
func (rc *Router) PostStop() error {
	if _, err := rc.session.Unsubscribe(&snslib.UnsubscribeInput{
		SubscriptionArn: rc.subscriptionArn,
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe. err: %+v", err)
	}
	return nil
}

// StartEventSource starts an SNS event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	log := listener.Log.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("operating on event source...")

	var snsEventSource *v1alpha1.SNSEventSource
	if err := yaml.Unmarshal(eventSource.Value, &snsEventSource); err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}

	return gwcommon.ProcessRoute(&Router{
		Route: &gwcommon.Route{
			Logger:      listener.Log,
			EventSource: eventSource,
			StartCh:     make(chan struct{}),
			Webhook:     snsEventSource.WebHook,
		},
		eventSource: snsEventSource,
		namespace:   listener.Namespace,
		k8sClient:   listener.K8sClient,
	}, controller, eventStream)
}
