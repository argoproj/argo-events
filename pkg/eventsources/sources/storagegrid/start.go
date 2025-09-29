/*
Copyright 2018 The Argoproj Authors.

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

package storagegrid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/argoproj/argo-events/pkg/eventsources/events"
	resty "github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/joncalhoun/qson"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// controller controls the webhook operations
var (
	controller = webhook.NewController()
)

var (
	respBody = `
<PublishResponse xmlns="http://argoevents-sns-server/">
    <PublishResult> 
        <MessageId>` + generateUUID().String() + `</MessageId> 
    </PublishResult> 
    <ResponseMetadata>
       <RequestId>` + generateUUID().String() + `</RequestId>
    </ResponseMetadata> 
</PublishResponse>` + "\n"

	notificationBodyTemplate = `
<NotificationConfiguration>
	<TopicConfiguration>
	  <Id>%s</Id>
	  <Topic>%s</Topic>
	  %s
	</TopicConfiguration>
</NotificationConfiguration>
` + "\n"
)

// set up the activation and inactivation channels to control the state of routes.
func init() {
	go webhook.ProcessRouteStatus(controller)
}

// generateUUID returns a new uuid
func generateUUID() uuid.UUID {
	return uuid.New()
}

// filterName filters object key based on configured prefix and/or suffix
func filterName(notification *events.StorageGridNotification, eventSource *v1alpha1.StorageGridEventSource) bool {
	if eventSource.Filter == nil {
		return true
	}
	if eventSource.Filter.Prefix != "" && eventSource.Filter.Suffix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, eventSource.Filter.Prefix) && strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, eventSource.Filter.Suffix)
	}
	if eventSource.Filter.Prefix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, eventSource.Filter.Prefix)
	}
	if eventSource.Filter.Suffix != "" {
		return strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, eventSource.Filter.Suffix)
	}
	return true
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
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.StorageGridEvent
}

// Implement Router
// 1. GetRoute
// 2. HandleRoute
// 3. PostActivate
// 4. PostDeactivate

// GetRoute returns the route
func (router *Router) GetRoute() *webhook.Route {
	return router.route
}

// HandleRoute handles new route
func (router *Router) HandleRoute(writer http.ResponseWriter, request *http.Request) {
	route := router.route

	logger := route.Logger.With(
		logging.LabelEndpoint, route.Context.Endpoint,
		logging.LabelPort, route.Context.Port,
		logging.LabelHTTPMethod, route.Context.Method,
	)

	logger.Info("processing incoming request...")

	if !route.Active {
		logger.Warn("endpoint is inactive, won't process the request")
		sharedutil.SendErrorResponse(writer, "inactive endpoint")
		return
	}

	logger.Info("parsing the request body...")
	request.Body = http.MaxBytesReader(writer, request.Body, route.Context.GetMaxPayloadSize())
	body, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Errorw("failed to parse request body", zap.Error(err))
		sharedutil.SendErrorResponse(writer, "")
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	if request.Method == http.MethodHead {
		respBody = ""
	}

	writer.WriteHeader(http.StatusOK)
	writer.Header().Add("Content-Type", "text/plain")
	if _, err := writer.Write([]byte(respBody)); err != nil {
		logger.Errorw("failed to write the response", zap.Error(err))
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	// notification received from storage grid is url encoded.
	parsedURL, err := url.QueryUnescape(string(body))
	if err != nil {
		logger.Errorw("failed to unescape request body url", zap.Error(err))
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}
	b, err := qson.ToJSON(parsedURL)
	if err != nil {
		logger.Errorw("failed to convert request body in JSON format", zap.Error(err))
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	logger.Info("converting request body to storage grid notification")
	var notification *events.StorageGridNotification
	err = json.Unmarshal(b, &notification)
	if err != nil {
		logger.Errorw("failed to convert the request body into storage grid notification", zap.Error(err))
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	if filterName(notification, router.storageGridEventSource) {
		defer func(start time.Time) {
			route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		logger.Info("new event received")
		eventData := &events.StorageGridEventData{
			Notification: notification,
			Metadata:     router.storageGridEventSource.Metadata,
		}
		eventBody, err := json.Marshal(eventData)
		if err != nil {
			logger.Errorw("failed to marshal the event data", zap.Error(err))
			route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
			return
		}
		webhook.DispatchEvent(route, eventBody, logger, writer)
		return
	}

	logger.Warn("discarding notification since it did not pass all filters")
}

// PostActivate performs operations once the route is activated and ready to consume requests
func (router *Router) PostActivate() error {
	eventSource := router.storageGridEventSource
	route := router.route

	authToken, err := sharedutil.GetSecretFromVolume(eventSource.AuthToken)
	if err != nil {
		return fmt.Errorf("AuthToken not found, %w", err)
	}

	registrationURL := sharedutil.FormattedURL(eventSource.Webhook.URL, eventSource.Webhook.Endpoint)

	httpClient := &http.Client{}

	if eventSource.TLS != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(eventSource.TLS)
		if err != nil {
			return fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		httpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	client := resty.NewWithClient(httpClient)

	logger := route.Logger.With(
		"registration-url", registrationURL,
		"bucket", eventSource.Bucket,
		"auth-secret-name", eventSource.AuthToken.Name,
		"api-url", eventSource.APIURL,
	)

	logger.Info("checking if the endpoint already exists...")

	response, err := client.R().
		SetHeader("Content-Type", v1alpha1.MediaTypeJSON).
		SetAuthToken(authToken).
		SetResult(&getEndpointResponse{}).
		SetError(&genericResponse{}).
		Get(sharedutil.FormattedURL(eventSource.APIURL, "/org/endpoints"))
	if err != nil {
		return err
	}

	if !response.IsSuccess() {
		errObj := response.Error().(*genericResponse)
		return fmt.Errorf("failed to list existing endpoints. reason: %s", errObj.Message.Text)
	}

	endpointResponse := response.Result().(*getEndpointResponse)

	isURNExists := false

	for _, endpoint := range endpointResponse.Data {
		if endpoint.EndpointURN == eventSource.TopicArn {
			logger.Info("endpoint with topic urn already exists, won't register duplicate endpoint")
			isURNExists = true
			break
		}
	}

	if !isURNExists {
		logger.Info("endpoint urn does not exist, registering a new endpoint")
		newEndpoint := createEndpointRequest{
			DisplayName: router.route.EventName,
			EndpointURI: sharedutil.FormattedURL(eventSource.Webhook.URL, eventSource.Webhook.Endpoint),
			EndpointURN: eventSource.TopicArn,
			AuthType:    "anonymous",
			InsecureTLS: true,
		}

		newEndpointBody, err := json.Marshal(&newEndpoint)
		if err != nil {
			return err
		}

		response, err := client.R().
			SetHeader("Content-Type", v1alpha1.MediaTypeJSON).
			SetAuthToken(authToken).
			SetBody(string(newEndpointBody)).
			SetResult(&genericResponse{}).
			SetError(&genericResponse{}).
			Post(sharedutil.FormattedURL(eventSource.APIURL, "/org/endpoints"))
		if err != nil {
			return err
		}

		if !response.IsSuccess() {
			errObj := response.Error().(*genericResponse)
			return fmt.Errorf("failed to register the endpoint. reason: %s", errObj.Message.Text)
		}

		logger.Info("successfully registered the endpoint")
	}

	logger.Info("registering notification configuration on storagegrid...")

	var events []string
	for _, event := range eventSource.Events {
		events = append(events, fmt.Sprintf("<Event>%s</Event>", event))
	}

	eventXML := strings.Join(events, "\n")

	notificationBody := fmt.Sprintf(notificationBodyTemplate, route.EventName, eventSource.TopicArn, eventXML)

	notification := &storageGridNotificationRequest{
		Notification: notificationBody,
	}

	notificationRequestBody, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	response, err = client.R().
		SetHeader("Content-Type", v1alpha1.MediaTypeJSON).
		SetAuthToken(authToken).
		SetBody(string(notificationRequestBody)).
		SetResult(&registerNotificationResponse{}).
		SetError(&genericResponse{}).
		Put(sharedutil.FormattedURL(eventSource.APIURL, fmt.Sprintf("/org/containers/%s/notification", eventSource.Bucket)))
	if err != nil {
		return err
	}

	if !response.IsSuccess() {
		errObj := response.Error().(*genericResponse)
		return fmt.Errorf("failed to configure notification. reason %s", errObj.Message.Text)
	}

	logger.Info("successfully registered notification configuration on storagegrid")
	return nil
}

// PostInactivate performs operations after the route is inactivated
func (router *Router) PostInactivate() error {
	return nil
}

// StartListening starts an event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the Storage Grid event source...")
	defer sources.Recover(el.GetEventName())

	storagegridEventSource := &el.StorageGridEventSource
	route := webhook.NewRoute(storagegridEventSource.Webhook, log, el.GetEventSourceName(), el.GetEventName(), el.Metrics)

	return webhook.ManageRoute(ctx, &Router{
		route:                  route,
		storageGridEventSource: storagegridEventSource,
	}, controller, dispatch)
}
