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

package gitlab

import (
	"bytes"
	"github.com/xanzy/go-gitlab"
	"io/ioutil"
	"net/http"
	"testing"

	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	rc = &Router{
		route:     gwcommon.GetFakeRoute(),
		clientset: fake.NewSimpleClientset(),
		namespace: "fake",
	}

	secretName     = "gitlab-access"
	accessKey      = "YWNjZXNz"
	LabelAccessKey = "accesskey"
)

func TestGetCredentials(t *testing.T) {
	convey.Convey("Given a kubernetes secret, get credentials", t, func() {
		secret, err := rc.clientset.CoreV1().Secrets(rc.namespace).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: rc.namespace,
			},
			Data: map[string][]byte{
				LabelAccessKey: []byte(accessKey),
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(secret, convey.ShouldNotBeNil)

		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		creds, err := rc.getCredentials(ps.(*gitlabEventSource).AccessToken)
		convey.So(err, convey.ShouldBeNil)
		convey.So(creds, convey.ShouldNotBeNil)
		convey.So(creds.token, convey.ShouldEqual, "YWNjZXNz")
	})
}

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		helper.ActiveEndpoints[rc.route.Webhook.Endpoint] = &gwcommon.Endpoint{
			DataCh: make(chan []byte),
		}

		convey.Convey("Inactive route should return error", func() {
			writer := &gwcommon.FakeHttpWriter{}
			ps, err := parseEventSource(es)
			convey.So(err, convey.ShouldBeNil)
			pbytes, err := yaml.Marshal(ps.(*gitlabEventSource))
			convey.So(err, convey.ShouldBeNil)
			rc.HandleRoute(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(pbytes)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)

			convey.Convey("Active route should return success", func() {
				helper.ActiveEndpoints[rc.route.Webhook.Endpoint].Active = true
				rc.hook = &gitlab.ProjectHook{
					URL:        "fake",
					PushEvents: true,
				}
				dataCh := make(chan []byte)
				go func() {
					resp := <-helper.ActiveEndpoints[rc.route.Webhook.Endpoint].DataCh
					dataCh <- resp
				}()

				rc.HandleRoute(writer, &http.Request{
					Body: ioutil.NopCloser(bytes.NewReader(pbytes)),
				})

				data := <-dataCh
				convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
				convey.So(string(data), convey.ShouldEqual, string(pbytes))
				rc.eventSource = ps.(*gitlabEventSource)
				err = rc.PostStart()
				convey.So(err, convey.ShouldNotBeNil)
			})
		})
	})
}
