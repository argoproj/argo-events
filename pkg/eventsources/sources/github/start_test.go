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

package github

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
)

var (
	router = &Router{
		route:             webhook.GetFakeRoute(),
		githubEventSource: &v1alpha1.GithubEventSource{},
	}
)

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		route := router.route
		route.DispatchChan = make(chan *webhook.Dispatch)

		convey.Convey("Inactive route should return error", func() {
			writer := &webhook.FakeHttpWriter{}
			githubEventSource := &v1alpha1.GithubEventSource{
				Webhook: &v1alpha1.WebhookContext{
					Endpoint: "/push",
					URL:      "http://webhook-gateway-svc",
					Port:     "12000",
				},
				Repositories: []v1alpha1.OwnedRepositories{
					{
						Owner: "fake",
						Names: []string{
							"fake0", "fake1",
						},
					},
				},
				Events: []string{
					"PushEvent",
				},
				APIToken: &corev1.SecretKeySelector{
					Key: "accessKey",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "github-access",
					},
				},
			}

			body, err := yaml.Marshal(githubEventSource)
			convey.So(err, convey.ShouldBeNil)

			router.HandleRoute(writer, &http.Request{
				Body: io.NopCloser(bytes.NewReader(body)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)

			convey.Convey("Active route should return success", func() {
				route.Active = true

				router.HandleRoute(writer, &http.Request{
					Body: io.NopCloser(bytes.NewReader(body)),
				})

				convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func TestRouteActiveHandlerDeprecated(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		route := router.route
		route.DispatchChan = make(chan *webhook.Dispatch)

		convey.Convey("Inactive route should return error", func() {
			writer := &webhook.FakeHttpWriter{}
			githubEventSource := &v1alpha1.GithubEventSource{
				Webhook: &v1alpha1.WebhookContext{
					Endpoint: "/push",
					URL:      "http://webhook-gateway-svc",
					Port:     "12000",
				},
				DeprecatedOwner:      "fake",
				DeprecatedRepository: "fake",
				Events: []string{
					"PushEvent",
				},
				APIToken: &corev1.SecretKeySelector{
					Key: "accessKey",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "github-access",
					},
				},
			}

			body, err := yaml.Marshal(githubEventSource)
			convey.So(err, convey.ShouldBeNil)

			router.HandleRoute(writer, &http.Request{
				Body: io.NopCloser(bytes.NewReader(body)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)

			convey.Convey("Active route should return success", func() {
				route.Active = true

				router.HandleRoute(writer, &http.Request{
					Body: io.NopCloser(bytes.NewReader(body)),
				})

				convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}

func TestAddEventTypeBody(t *testing.T) {
	convey.Convey("Given a request", t, func() {
		var (
			buf        = bytes.NewBuffer([]byte(`{ "hello": "world" }`))
			eventType  = "PushEvent"
			deliveryID = "131C7C9B-A571-4F60-9ACA-EA3ADA19FABE"
		)
		request, err := http.NewRequest("POST", "http://example.com", buf)
		convey.So(err, convey.ShouldBeNil)
		request.Header.Set("X-GitHub-Event", eventType)
		request.Header.Set("X-GitHub-Delivery", deliveryID)
		request.Header.Set("Content-Type", "application/json")

		convey.Convey("Delivery headers should be written to message", func() {
			body, err := parseValidateRequest(request, []byte{})
			convey.So(err, convey.ShouldBeNil)
			payload := make(map[string]interface{})
			err = json.Unmarshal(body, &payload)
			convey.So(err, convey.ShouldBeNil)
			convey.So(err, convey.ShouldBeNil)
			convey.So(payload["X-GitHub-Event"], convey.ShouldEqual, eventType)
			convey.So(payload["X-GitHub-Delivery"], convey.ShouldEqual, deliveryID)
		})
	})
}
