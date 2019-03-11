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

package github

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/argoproj/argo-events/common"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	"github.com/google/go-github/github"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	ese = &GithubEventSourceExecutor{
		Clientset: fake.NewSimpleClientset(),
		Namespace: "fake",
		Log:       common.GetLoggerContext(common.LoggerConf()).Logger(),
	}

	secretName     = "githab-access"
	accessKey      = "YWNjZXNz"
	LabelAccessKey = "accesskey"
)

func TestGetCredentials(t *testing.T) {
	convey.Convey("Given a kubernetes secret, get credentials", t, func() {
		secret, err := ese.Clientset.CoreV1().Secrets(ese.Namespace).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: ese.Namespace,
			},
			Data: map[string][]byte{
				LabelAccessKey: []byte(accessKey),
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(secret, convey.ShouldNotBeNil)

		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		creds, err := ese.getCredentials(ps.(*githubConfig).APIToken)
		convey.So(err, convey.ShouldBeNil)
		convey.So(creds, convey.ShouldNotBeNil)
		convey.So(creds.secret, convey.ShouldEqual, "YWNjZXNz")
	})
}

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		rc := gwcommon.GetFakeRouteConfig()
		helper.ActiveEndpoints[rc.Webhook.Endpoint] = &gwcommon.Endpoint{
			DataCh: make(chan []byte),
		}

		convey.Convey("Inactive route should return error", func() {
			writer := &gwcommon.FakeHttpWriter{}
			ps, err := parseEventSource(es)
			convey.So(err, convey.ShouldBeNil)
			pbytes, err := yaml.Marshal(ps.(*githubConfig))
			convey.So(err, convey.ShouldBeNil)
			RouteActiveHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(pbytes)),
			}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)

			convey.Convey("Active route should return success", func() {
				helper.ActiveEndpoints[rc.Webhook.Endpoint].Active = true
				rc.Configs[labelWebhook] = &github.Hook{
					Config: make(map[string]interface{}),
				}
				dataCh := make(chan []byte)
				go func() {
					resp := <-helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh
					dataCh <- resp
				}()

				RouteActiveHandler(writer, &http.Request{
					Body: ioutil.NopCloser(bytes.NewReader(pbytes)),
				}, rc)

				data := <-dataCh
				convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
				convey.So(string(data), convey.ShouldEqual, string(pbytes))
				rc.Configs[labelGithubConfig] = ps.(*githubConfig)
				err = ese.PostActivate(rc)
				convey.So(err, convey.ShouldNotBeNil)
			})
		})
	})
}

func TestValidatePayload(t *testing.T) {
	convey.Convey("Given a secret and body, validate payload", t, func() {
		req := http.Request{
			Header: make(map[string][]string),
		}
		req.Header.Set(githubSignatureHeader, "fake-sig")
		req.Header.Set(githubEventHeader, "fake-event")
		req.Header.Set(githubDeliveryHeader, "fake-delivery")
		err := validatePayload([]byte("fake-secret"), req.Header, []byte("fake-body"))
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(err.Error(), convey.ShouldEqual, "invalid signature")
	})
}

func TestVerifySignature(t *testing.T) {
	convey.Convey("Given a fake signature, validate payload", t, func() {
		ok := verifySignature([]byte("fake-secret"), "fake-sig", []byte("fake-body"))
		convey.So(ok, convey.ShouldEqual, false)
	})
}
