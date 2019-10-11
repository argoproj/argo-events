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
	"bytes"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"io/ioutil"
	"net/http"
	"testing"

	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/aws/aws-sdk-go/aws/credentials"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAWSSNS(t *testing.T) {
	convey.Convey("Given an route configuration", t, func() {
		rc := &RouteConfig{
			Route:     gwcommon.GetFakeRoute(),
			namespace: "fake",
			clientset: fake.NewSimpleClientset(),
		}
		r := rc.Route

		helper.ActiveEndpoints[r.Webhook.Endpoint] = &gwcommon.Endpoint{
			DataCh: make(chan []byte),
		}
		writer := &gwcommon.FakeHttpWriter{}
		subscriptionArn := "arn://fake"
		awsSession, err := gwcommon.GetAWSSession(credentials.NewStaticCredentialsFromCreds(credentials.Value{
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
		}), "mock-region")

		convey.So(err, convey.ShouldBeNil)

		snsSession := snslib.New(awsSession)
		rc.session = snsSession
		rc.subscriptionArn = &subscriptionArn

		convey.Convey("handle the inactive route", func() {
			rc.RouteHandler(writer, &http.Request{})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)

		helper.ActiveEndpoints[r.Webhook.Endpoint].Active = true
		rc.snses = ps.(*v1alpha1.SNSEventSource)

		convey.Convey("handle the active route", func() {
			payload := httpNotification{
				TopicArn: "arn://fake",
				Token:    "faketoken",
				Type:     messageTypeSubscriptionConfirmation,
			}

			payloadBytes, err := yaml.Marshal(payload)
			convey.So(err, convey.ShouldBeNil)
			rc.RouteHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewBuffer(payloadBytes)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)

			dataCh := make(chan []byte)

			go func() {
				data := <-helper.ActiveEndpoints[r.Webhook.Endpoint].DataCh
				dataCh <- data
			}()

			payload.Type = messageTypeNotification
			payloadBytes, err = yaml.Marshal(payload)
			convey.So(err, convey.ShouldBeNil)
			rc.RouteHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewBuffer(payloadBytes)),
			})
			data := <-dataCh
			convey.So(data, convey.ShouldNotBeNil)
		})

		convey.Convey("Run post activate", func() {
			err := rc.PostStart()
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("Run post stop", func() {
			err = rc.PostStop()
			convey.So(err, convey.ShouldNotBeNil)
		})

		psWithoutCreds, err2 := parseEventSource(esWithoutCreds)
		convey.So(err2, convey.ShouldBeNil)

		rc.snses = psWithoutCreds.(*v1alpha1.SNSEventSource)

		convey.Convey("Run post activate on event source without credentials", func() {
			err := rc.PostStart()
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}
