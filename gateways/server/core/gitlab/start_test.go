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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	"github.com/xanzy/go-gitlab"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	secretName     = "gitlab-access"
	accessKey      = "YWNjZXNz"
	LabelAccessKey = "accesskey"

	router = &Router{
		route:     webhook.GetFakeRoute(),
		k8sClient: fake.NewSimpleClientset(),
		gitlabEventSource: &v1alpha1.GitlabEventSource{
			Namespace: "fake",
			AccessToken: &corev1.SecretKeySelector{
				Key: LabelAccessKey,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
			},
		},
	}
)

func TestGetCredentials(t *testing.T) {
	convey.Convey("Given a kubernetes secret, get credentials", t, func() {
		gitlabEventSource := router.gitlabEventSource
		secret, err := router.k8sClient.CoreV1().Secrets(gitlabEventSource.Namespace).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			Data: map[string][]byte{
				LabelAccessKey: []byte(accessKey),
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(secret, convey.ShouldNotBeNil)

		creds, err := router.getCredentials(gitlabEventSource.AccessToken, gitlabEventSource.Namespace)
		convey.So(err, convey.ShouldBeNil)
		convey.So(creds, convey.ShouldNotBeNil)
		convey.So(creds.token, convey.ShouldEqual, "YWNjZXNz")
	})
}

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		convey.Convey("Inactive route should return error", func() {
			writer := &webhook.FakeHttpWriter{}
			gitlabEventSource := router.gitlabEventSource
			route := router.route

			pbytes, err := yaml.Marshal(gitlabEventSource)
			convey.So(err, convey.ShouldBeNil)
			router.HandleRoute(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(pbytes)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)

			convey.Convey("Active route should return success", func() {
				route.Active = true
				router.hook = &gitlab.ProjectHook{
					URL:        "fake",
					PushEvents: true,
				}
				dataCh := make(chan []byte)
				go func() {
					resp := <-route.DataCh
					dataCh <- resp
				}()

				router.HandleRoute(writer, &http.Request{
					Body: ioutil.NopCloser(bytes.NewReader(pbytes)),
				})

				data := <-dataCh
				convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
				convey.So(string(data), convey.ShouldEqual, string(pbytes))
				err = router.PostInactivate()
				convey.So(err, convey.ShouldNotBeNil)
			})
		})
	})
}
