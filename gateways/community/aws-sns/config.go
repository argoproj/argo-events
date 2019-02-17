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
	"k8s.io/client-go/kubernetes"
	"time"

	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
)

const (
	MESSAGE_TYPE_SUBSCRIPTION_CONFIRMATION = "SubscriptionConfirmation"
	MESSAGE_TYPE_UNSUBSCRIBE_CONFIRMATION  = "UnsubscribeConfirmation"
	MESSAGE_TYPE_NOTIFICATION              = "Notification"
)

var (
	snsProtocol = "http"
)

// SNSEventSourceExecutor implements Eventing
type SNSEventSourceExecutor struct {
	Log zerolog.Logger
	// Clientset is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
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

// snsConfig contains configuration to subscribe to SNS topic
type snsConfig struct {
	TopicArn string `json:"topicArn"`
	// Endpoint you wish to register
	Endpoint string `json:"endpoint"`
	// Port on which http server is running.
	Port      string                    `json:"port"`
	AccessKey *corev1.SecretKeySelector `json:"accessKey" protobuf:"bytes,5,opt,name=accessKey"`
	SecretKey *corev1.SecretKeySelector `json:"secretKey" protobuf:"bytes,6,opt,name=secretKey"`
	Region    string                    `json:"region"`
}

func parseEventSource(es string) (*snsConfig, error) {
	var n *snsConfig
	err := yaml.Unmarshal([]byte(es), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
