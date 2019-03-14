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
	"github.com/argoproj/argo-events/common"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/aws/aws-sdk-go/aws/credentials"
	snslib "github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"testing"
)

func TestAWSSNS(t *testing.T) {
	convey.Convey("Given an route configuration", t, func() {
		rc := gwcommon.GetFakeRouteConfig()
		helper.ActiveEndpoints[rc.Webhook.Endpoint] = &gwcommon.Endpoint{
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
		rc.Configs[labelSNSSession] = snsSession
		rc.Configs[labelSubscriptionArn] = &subscriptionArn

		convey.Convey("handle the inactive route", func() {
			RouteActiveHandler(writer, &http.Request{}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)

		helper.ActiveEndpoints[rc.Webhook.Endpoint].Active = true
		rc.Configs[labelSNSConfig] = ps.(*snsConfig)

		convey.Convey("handle the active route", func() {
			payload := httpNotification{
				TopicArn: "arn://fake",
				Token:    "faketoken",
				Type:     messageTypeSubscriptionConfirmation,
			}

			payloadBytes, err := yaml.Marshal(payload)
			convey.So(err, convey.ShouldBeNil)
			RouteActiveHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewBuffer(payloadBytes)),
			}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
			convey.So(string(writer.Payload), convey.ShouldEqual, "failed to confirm subscription")

			go func() {
				<-helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh
			}()

			payload.Type = messageTypeNotification
			payloadBytes, err = yaml.Marshal(payload)
			convey.So(err, convey.ShouldBeNil)
			RouteActiveHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewBuffer(payloadBytes)),
			}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
		})

		convey.Convey("Run post activate", func() {
			ese := SNSEventSourceExecutor{
				Namespace: "fake",
				Clientset: fake.NewSimpleClientset(),
				Log:       common.GetLoggerContext(common.LoggerConf()).Logger(),
			}
			err := ese.PostActivate(rc)
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("Run post stop", func() {
			err = PostStop(rc)
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}
