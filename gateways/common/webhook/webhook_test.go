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

package webhook

import (
	"net/http"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
)

var Hook = &Context{
	Endpoint: "/fake",
	Port:     "12000",
	URL:      "test-url",
}

type FakeHttpWriter struct {
	HeaderStatus int
	Payload      []byte
}

func (f *FakeHttpWriter) Header() http.Header {
	return http.Header{}
}

func (f *FakeHttpWriter) Write(body []byte) (int, error) {
	f.Payload = body
	return len(body), nil
}

func (f *FakeHttpWriter) WriteHeader(status int) {
	f.HeaderStatus = status
}

type FakeRouter struct {
	route *Route
}

func (f *FakeRouter) GetRoute() *Route {
	return f.route
}

func (f *FakeRouter) HandleRoute(writer http.ResponseWriter, request *http.Request) {
}

func (f *FakeRouter) PostActivate() error {
	return nil
}

func (f *FakeRouter) PostInactivate() error {
	return nil
}

func GetFakeRoute() *Route {
	logger := common.NewArgoEventsLogger()
	return &Route{
		Context: Hook,
		EventSource: &gateways.EventSource{
			Name:  "fake-event-source",
			Value: []byte("hello"),
			Id:    "123",
		},
		Logger:  logger,
		StartCh: make(chan struct{}),
	}
}

var rc = &FakeRouter{
	route: GetFakeRoute(),
}

func TestValidateWebhook(t *testing.T) {
	convey.Convey("Given a webhook, validate it", t, func() {
		convey.So(ValidateWebhookContext(Hook), convey.ShouldBeNil)
	})
}

func TestNewWebhookHelper(t *testing.T) {
	convey.Convey("Make sure webhook helper is not empty", t, func() {
		controller := NewController()
		convey.So(controller, convey.ShouldNotBeNil)
	})
}
