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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	commonaws "github.com/argoproj/argo-events/gateways/server/common/aws"
	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
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

	logger := route.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: route.EventSource.Name,
			common.LabelEndpoint:    route.Context.Endpoint,
			common.LabelPort:        route.Context.Port,
			common.LabelHTTPMethod:  route.Context.Method,
		})

	logger.Info("request received from event source")

	if !route.Active {
		logger.Info("endpoint is not active, won't process the request")
		common.SendErrorResponse(writer, "inactive endpoint")
		return
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.WithError(err).Error("failed to parse the request body")
		common.SendErrorResponse(writer, err.Error())
		return
	}

	var notification *httpNotification
	err = yaml.Unmarshal(body, &notification)
	if err != nil {
		logger.WithError(err).Error("failed to convert request payload into sns notification")
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
			logger.WithError(err).Error("failed to send confirmation response to aws sns")
			common.SendErrorResponse(writer, err.Error())
			return
		}

		logger.Infoln("subscription successfully confirmed to aws sns")
		router.subscriptionArn = response.SubscriptionArn

	case messageTypeNotification:
		logger.Infoln("dispatching notification on route's data channel")

		eventData := &events.SNSEventData{Body: body}
		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			logger.WithError(err).Error("failed to marshal the event data")
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

	logger := route.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: route.EventSource.Name,
			common.LabelEndpoint:    route.Context.Endpoint,
			common.LabelPort:        route.Context.Port,
			common.LabelHTTPMethod:  route.Context.Method,
			"topic-arn":             router.eventSource.TopicArn,
		})

	// In order to successfully subscribe to sns topic,
	// 1. Fetch credentials if configured explicitly. Users can use something like https://github.com/jtblin/kube2iam
	//    which will help not configure creds explicitly.
	// 2. Get AWS session
	// 3. Subscribe to a topic

	logger.Info("subscribing to sns topic...")

	snsEventSource := router.eventSource

	awsSession, err := commonaws.CreateAWSSession(router.k8sClient, snsEventSource.Namespace, snsEventSource.Region, snsEventSource.AccessKey, snsEventSource.SecretKey)
	if err != nil {
		return err
	}

	router.session = snslib.New(awsSession)
	formattedUrl := common.FormattedURL(snsEventSource.Webhook.URL, snsEventSource.Webhook.Endpoint)
	if _, err := router.session.Subscribe(&snslib.SubscribeInput{
		Endpoint: &formattedUrl,
		Protocol: func(endpoint string) *string {
			Protocol := "http"
			if matched, _ := regexp.MatchString(`https://.*`, endpoint); matched {
				Protocol = "https"
				return &Protocol
			}
			return &Protocol
		}(formattedUrl),
		TopicArn: &snsEventSource.TopicArn,
	}); err != nil {
		return err
	}

	return nil
}

// PostInactive refers to operations performed after a route is successfully inactivated
func (router *Router) PostInactivate() error {
	// After event source is removed, the subscription is cancelled.
	if _, err := router.session.Unsubscribe(&snslib.UnsubscribeInput{
		SubscriptionArn: router.subscriptionArn,
	}); err != nil {
		return err
	}
	return nil
}

// StartEventSource starts an SNS event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer server.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)
	logger.Info("started processing the event source...")

	var snsEventSource *v1alpha1.SNSEventSource
	if err := yaml.Unmarshal(eventSource.Value, &snsEventSource); err != nil {
		logger.WithError(err).Error("failed to parse event source")
		return err
	}

	route := webhook.NewRoute(snsEventSource.Webhook, listener.Logger, eventSource)

	logger.Infoln("operating on the route...")
	return webhook.ManageRoute(&Router{
		Route:       route,
		eventSource: snsEventSource,
		k8sClient:   listener.K8sClient,
	}, controller, eventStream)
}
