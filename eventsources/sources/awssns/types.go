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
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/argoproj/argo-events/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	snslib "github.com/aws/aws-sdk-go/service/sns"
)

const (
	messageTypeSubscriptionConfirmation = "SubscriptionConfirmation"
	messageTypeNotification             = "Notification"
)

// Router contains information for a route
type Router struct {
	// Route contains webhook context and configuration related to api route
	Route *webhook.Route
	// eventSource refers to sns event source configuration
	eventSource *v1alpha1.SNSEventSource
	// session refers to aws session
	session *snslib.SNS
	// subscriptionArn is sns arn
	subscriptionArn *string
}

// Json http notifications
// SNS posts those to your http url endpoint if http is selected as delivery method.
// http://docs.aws.amazon.com/sns/latest/dg/json-formats.html#http-subscription-confirmation-json
// http://docs.aws.amazon.com/sns/latest/dg/json-formats.html#http-notification-json
// http://docs.aws.amazon.com/sns/latest/dg/json-formats.html#http-unsubscribe-confirmation-json
type httpNotification struct {
	Type             string    `json:"Type"`
	MessageID        string    `json:"MessageId"`
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

func (m *httpNotification) verify() error {
	msgSig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil {
		return errors.Wrap(err, "failed to base64 decode signature")
	}

	res, err := http.Get(m.SigningCertURL)
	if err != nil {
		return errors.Wrap(err, "failed to fetch signing cert")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read signing cert body")
	}

	p, _ := pem.Decode(body)
	if p == nil {
		return errors.New("nothing found in pem encoded bytes")
	}

	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse signing cert")
	}

	err = cert.CheckSignature(x509.SHA1WithRSA, m.sigSerialized(), msgSig)
	if err != nil {
		return errors.Wrap(err, "message signature check error")
	}

	return nil
}

func (m *httpNotification) sigSerialized() []byte {
	buf := &bytes.Buffer{}
	v := reflect.ValueOf(m)
	snsSigKeys := map[string][]string{}
	snsKeyRealNames := map[string]string{
		"MessageID": "MessageId",
		"TopicARN":  "TopicArn",
	}

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
