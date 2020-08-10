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

package awssns

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	commonaws "github.com/argoproj/argo-events/eventsources/common/aws"
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
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
func (router *Router) GetRoute() *webhook.Route {
	return router.Route
}

// HandleRoute handles new routes
func (router *Router) HandleRoute(writer http.ResponseWriter, request *http.Request) {
	route := router.Route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
	).Desugar()

	logger.Info("request received from event source")

	if !route.Active {
		logger.Info("endpoint is not active, won't process the request")
		common.SendErrorResponse(writer, "inactive endpoint")
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error("failed to parse the request body", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
		return
	}

	var notification *httpNotification
	err = yaml.Unmarshal(body, &notification)
	if err != nil {
		logger.Error("failed to convert request payload into sns notification", zap.Error(err))
		common.SendErrorResponse(writer, err.Error())
		return
	}

	switch notification.Type {
	case messageTypeSubscriptionConfirmation:
		awsSession := router.session

		response, err := awsSession.ConfirmSubscription(&snslib.ConfirmSubscriptionInput{
			TopicArn: &router.eventSource.TopicArn,
			Token:    &notification.Token,
		})
		if err != nil {
			logger.Error("failed to send confirmation response to aws sns", zap.Error(err))
			common.SendErrorResponse(writer, err.Error())
			return
		}

		logger.Info("subscription successfully confirmed to aws sns")
		router.subscriptionArn = response.SubscriptionArn

	case messageTypeNotification:
		logger.Info("dispatching notification on route's data channel")

		eventData := &events.SNSEventData{
			Header:   request.Header,
			Body:     (*json.RawMessage)(&body),
			Metadata: router.eventSource.Metadata,
		}

		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			logger.Error("failed to marshal the event data", zap.Error(err))
			common.SendErrorResponse(writer, err.Error())
			return
		}
		route.DataCh <- eventBytes
	}

	logger.Info("request has been successfully processed")
}

// PostActivate refers to operations performed after a route is successfully activated
func (router *Router) PostActivate() error {
	route := router.Route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
		"topic-arn", router.eventSource.TopicArn,
	)

	// In order to successfully subscribe to sns topic,
	// 1. Fetch credentials if configured explicitly. Users can use something like https://github.com/jtblin/kube2iam
	//    which will help not configure creds explicitly.
	// 2. Get AWS session
	// 3. Subscribe to a topic

	logger.Info("subscribing to sns topic...")

	snsEventSource := router.eventSource

	awsSession, err := commonaws.CreateAWSSessionWithCredsInVolume(snsEventSource.Region, snsEventSource.RoleARN, snsEventSource.AccessKey, snsEventSource.SecretKey)
	if err != nil {
		return err
	}

	router.session = snslib.New(awsSession)
	formattedURL := common.FormattedURL(snsEventSource.Webhook.URL, snsEventSource.Webhook.Endpoint)
	if _, err := router.session.Subscribe(&snslib.SubscribeInput{
		Endpoint: &formattedURL,
		Protocol: func(endpoint string) *string {
			Protocol := "http"
			if matched, _ := regexp.MatchString(`https://.*`, endpoint); matched {
				Protocol = "https"
				return &Protocol
			}
			return &Protocol
		}(formattedURL),
		TopicArn: &snsEventSource.TopicArn,
	}); err != nil {
		return err
	}

	return nil
}

// PostInactivate refers to operations performed after a route is successfully inactivated
func (router *Router) PostInactivate() error {
	// After event source is removed, the subscription is cancelled.
	if _, err := router.session.Unsubscribe(&snslib.UnsubscribeInput{
		SubscriptionArn: router.subscriptionArn,
	}); err != nil {
		return err
	}
	return nil
}

// EventListener implements Eventing for aws sns event source
type EventListener struct {
	EventSourceName string
	EventName       string
	SNSEventSource  v1alpha1.SNSEventSource
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.SNSEvent
}

// StartListening starts an SNS event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	logger := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	defer sources.Recover(el.GetEventName())

	logger.Info("started processing the AWS SNS event source...")

	route := webhook.NewRoute(el.SNSEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName())

	logger.Info("operating on the route...")
	return webhook.ManageRoute(ctx, &Router{
		Route:       route,
		eventSource: &el.SNSEventSource,
	}, controller, dispatch)
}
