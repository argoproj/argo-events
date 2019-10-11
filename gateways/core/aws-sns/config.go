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
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/sirupsen/logrus"
	"time"

	gwcommon "github.com/argoproj/argo-events/gateways/common"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
	"k8s.io/client-go/kubernetes"
)

const ArgoEventsEventSourceVersion = "v0.10"

const (
	messageTypeSubscriptionConfirmation = "SubscriptionConfirmation"
	messageTypeNotification             = "Notification"
)

var (
	snsProtocol = "http"
)

// SNSEventSourceExecutor implements Eventing
type SNSEventSourceExecutor struct {
	Log *logrus.Logger
	// Clientset is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// RouteConfig contains information for a route
type RouteConfig struct {
	Route           *gwcommon.Route
	snses           *v1alpha1.SNSEventSource
	session         *snslib.SNS
	subscriptionArn *string
	clientset       kubernetes.Interface
	namespace       string
}

// Json http notifications
// SNS posts those to your http url endpoint if http is selected as delivery method.
// http://docs.aws.amazon.com/sns/latest/dg/json-formats.html#http-subscription-confirmation-json
// http://docs.aws.amazon.com/sns/latest/dg/json-formats.html#http-notification-json
// http://docs.aws.amazon.com/sns/latest/dg/json-formats.html#http-unsubscribe-confirmation-json
type httpNotification struct {
	Type             string    `json:"Type"`
	MessageId        string    `json:"MessageId"`
	Token            string    `json:"Token,omitempty"` // Only for subscribe and unsubscribe
	TopicArn         string    `json:"TopicArn"`
	Subject          string    `json:"Subject,omitempty"` // Only for Notification
	Message          string    `json:"Message"`
	SubscribeURL     string    `json:"SubscribeURL,omitempty"` // Only for subscribe and unsubscribe
	Timestamp        time.Time `json:"Timestamp"`
	SignatureVersion string    `json:"SignatureVersion"`
	Signature        string    `json:"Signature"`
	SigningCertURL   string    `json:"SigningCertURL"`
	UnsubscribeURL   string    `json:"UnsubscribeURL,omitempty"` // Only for notifications
}

func parseEventSource(es string) (interface{}, error) {
	var ses *v1alpha1.SNSEventSource
	err := yaml.Unmarshal([]byte(es), &ses)
	if err != nil {
		return nil, err
	}
	return ses, nil
}
