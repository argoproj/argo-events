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
	"io/ioutil"
	"net/http"
	"testing"

	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/aws/aws-sdk-go/aws/credentials"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAWSSNS(t *testing.T) {
	convey.Convey("Given an route configuration", t, func() {
		rc := &Router{
			Route:     gwcommon.GetFakeRoute(),
			namespace: "fake",
			k8sClient: fake.NewSimpleClientset(),
		}
		r := rc.Route

		controller.ActiveEndpoints[r.Webhook.Endpoint] = &gwcommon.Endpoint{
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
			rc.HandleRoute(writer, &http.Request{})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		snsEventSource := &v1alpha1.SNSEventSource{
			WebHook: &gwcommon.Webhook{
				Endpoint: "/test",
				Port:     "8080",
				URL:      "myurl/test",
			},
			Region:   "us-east-1",
			TopicArn: "test-arn",
			AccessKey: &corev1.SecretKeySelector{
				Key: "accesskey",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "sns",
				},
			},
			SecretKey: &corev1.SecretKeySelector{
				Key: "secretkey",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "sns",
				},
			},
		}

		controller.ActiveEndpoints[r.Webhook.Endpoint].Active = true
		rc.eventSource = snsEventSource

		convey.Convey("handle the active route", func() {
			payload := httpNotification{
				TopicArn: "arn://fake",
				Token:    "faketoken",
				Type:     messageTypeSubscriptionConfirmation,
			}

			payloadBytes, err := yaml.Marshal(payload)
			convey.So(err, convey.ShouldBeNil)
			rc.HandleRoute(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewBuffer(payloadBytes)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)

			dataCh := make(chan []byte)

			go func() {
				data := <-controller.ActiveEndpoints[r.Webhook.Endpoint].DataCh
				dataCh <- data
			}()

			payload.Type = messageTypeNotification
			payloadBytes, err = yaml.Marshal(payload)
			convey.So(err, convey.ShouldBeNil)
			rc.HandleRoute(writer, &http.Request{
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

		rc.eventSource = &v1alpha1.SNSEventSource{
			WebHook: &gwcommon.Webhook{
				Endpoint: "/test",
				Port:     "8080",
				URL:      "myurl/test",
			},
			TopicArn: "test-arn",
			Region:   "us-east-1",
		}

		convey.Convey("Run post activate on event source without credentials", func() {
			err := rc.PostStart()
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}
