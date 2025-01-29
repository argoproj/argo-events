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

package awssns

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
	"go.uber.org/zap"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	commonaws "github.com/argoproj/argo-events/pkg/eventsources/common/aws"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

var (
	// controller controls the webhook operations
	controller = webhook.NewController()

	// used for SNS verification
	snsSigKeys      = map[string][]string{}
	snsKeyRealNames = map[string]string{
		"MessageID": "MessageId",
		"TopicARN":  "TopicArn",
	}
)

// set up route activation and deactivation channels
func init() {
	go webhook.ProcessRouteStatus(controller)

	snsSigKeys[messageTypeNotification] = []string{
		"Message",
		"MessageID",
		"Subject",
		"Timestamp",
		"TopicARN",
		"Type",
	}
	snsSigKeys[messageTypeSubscriptionConfirmation] = []string{
		"Message",
		"MessageID",
		"SubscribeURL",
		"Timestamp",
		"Token",
		"TopicARN",
		"Type",
	}
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
	)

	logger.Info("request received from event source")

	if !route.Active {
		logger.Info("endpoint is not active, won't process the request")
		sharedutil.SendErrorResponse(writer, "inactive endpoint")
		return
	}

	defer func(start time.Time) {
		route.Metrics.EventProcessingDuration(route.EventSourceName, route.EventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	request.Body = http.MaxBytesReader(writer, request.Body, route.Context.GetMaxPayloadSize())
	body, err := io.ReadAll(request.Body)
	if err != nil {
		logger.Errorw("failed to parse the request body", zap.Error(err))
		sharedutil.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	var notification *httpNotification
	err = yaml.Unmarshal(body, &notification)
	if err != nil {
		logger.Errorw("failed to convert request payload into sns notification", zap.Error(err))
		sharedutil.SendErrorResponse(writer, err.Error())
		route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
		return
	}

	if notification == nil {
		sharedutil.SendErrorResponse(writer, "bad request, not a valid SNS notification")
		return
	}

	// SNS Signature Verification
	if router.eventSource.ValidateSignature {
		err = notification.verify()
		if err != nil {
			logger.Errorw("failed to verify sns message", zap.Error(err))
			sharedutil.SendErrorResponse(writer, err.Error())
			route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
			return
		}
	}

	switch notification.Type {
	case messageTypeSubscriptionConfirmation:
		awsSession := router.session

		response, err := awsSession.ConfirmSubscription(&snslib.ConfirmSubscriptionInput{
			TopicArn: &router.eventSource.TopicArn,
			Token:    &notification.Token,
		})
		if err != nil {
			logger.Errorw("failed to send confirmation response to aws sns", zap.Error(err))
			sharedutil.SendErrorResponse(writer, err.Error())
			route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
			return
		}

		logger.Info("subscription successfully confirmed to aws sns")
		router.subscriptionArn = response.SubscriptionArn

	case messageTypeNotification:

		eventData := &events.SNSEventData{
			Header:   request.Header,
			Body:     (*json.RawMessage)(&body),
			Metadata: router.eventSource.Metadata,
		}

		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			logger.Errorw("failed to marshal the event data", zap.Error(err))
			sharedutil.SendErrorResponse(writer, err.Error())
			route.Metrics.EventProcessingFailed(route.EventSourceName, route.EventName)
			return
		}
		webhook.DispatchEvent(route, eventBytes, logger, writer)
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

	awsSession, err := commonaws.CreateAWSSessionWithCredsInVolume(snsEventSource.Region, snsEventSource.RoleARN, snsEventSource.AccessKey, snsEventSource.SecretKey, nil)
	if err != nil {
		return err
	}

	if snsEventSource.Endpoint == "" {
		router.session = snslib.New(awsSession)
	} else {
		router.session = snslib.New(awsSession, &aws.Config{Endpoint: &snsEventSource.Endpoint, Region: &snsEventSource.Region})
	}

	formattedURL := sharedutil.FormattedURL(snsEventSource.Webhook.URL, snsEventSource.Webhook.Endpoint)
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
	SNSEventSource  aev1.SNSEventSource
	Metrics         *metrics.Metrics
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
func (el *EventListener) GetEventSourceType() aev1.EventSourceType {
	return aev1.SNSEvent
}

// StartListening starts an SNS event source
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	logger := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	defer sources.Recover(el.GetEventName())

	logger.Info("started processing the AWS SNS event source...")

	route := webhook.NewRoute(el.SNSEventSource.Webhook, logger, el.GetEventSourceName(), el.GetEventName(), el.Metrics)

	logger.Info("operating on the route...")
	return webhook.ManageRoute(ctx, &Router{
		Route:       route,
		eventSource: &el.SNSEventSource,
	}, controller, dispatch)
}

func (m *httpNotification) verifySigningCertUrl() error {
	regexSigningCertHost := `^sns\.[a-zA-Z0-9\-]{3,}\.amazonaws\.com(\.cn)?$`
	regex := regexp.MustCompile(regexSigningCertHost)
	url, err := url.Parse(m.SigningCertURL)
	if err != nil {
		return fmt.Errorf("SigningCertURL is not a valid URL, %w", err)
	}
	if !regex.MatchString(url.Hostname()) {
		return fmt.Errorf("SigningCertURL hostname `%s` does not match `%s`", url.Hostname(), regexSigningCertHost)
	}
	if url.Scheme != "https" {
		return fmt.Errorf("SigningCertURL is not using https")
	}
	return nil
}

func (m *httpNotification) verify() error {
	msgSig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil {
		return fmt.Errorf("failed to base64 decode signature, %w", err)
	}

	if err := m.verifySigningCertUrl(); err != nil {
		return fmt.Errorf("failed to verify SigningCertURL, %w", err)
	}

	res, err := http.Get(m.SigningCertURL)
	if err != nil {
		return fmt.Errorf("failed to fetch signing cert, %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 65*1024))
	if err != nil {
		return fmt.Errorf("failed to read signing cert body, %w", err)
	}

	p, _ := pem.Decode(body)
	if p == nil {
		return fmt.Errorf("nothing found in pem encoded bytes")
	}

	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse signing cert, %w", err)
	}

	err = cert.CheckSignature(x509.SHA1WithRSA, m.sigSerialized(), msgSig)
	if err != nil {
		return fmt.Errorf("message signature check error, %w", err)
	}

	return nil
}

func (m *httpNotification) sigSerialized() []byte {
	buf := &bytes.Buffer{}
	v := reflect.ValueOf(m)

	for _, key := range snsSigKeys[m.Type] {
		field := reflect.Indirect(v).FieldByName(key)
		val := field.String()
		if !field.IsValid() || val == "" {
			continue
		}
		if rn, ok := snsKeyRealNames[key]; ok {
			key = rn
		}
		buf.WriteString(key + "\n")
		buf.WriteString(val + "\n")
	}

	return buf.Bytes()
}
