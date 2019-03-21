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

package common

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

var rc = &FakeRouteConfig{
	route: GetFakeRoute(),
}

func TestProcessRoute(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		convey.Convey("Activate the route configuration", func() {

			rc.route.Webhook.mux = http.NewServeMux()

			ctx, cancel := context.WithCancel(context.Background())
			fgs := &FakeGRPCStream{
				Ctx: ctx,
			}

			helper := NewWebhookHelper()
			helper.ActiveEndpoints[rc.route.Webhook.Endpoint] = &Endpoint{
				DataCh: make(chan []byte),
			}
			helper.ActiveServers[rc.route.Webhook.Port] = &activeServer{
				errChan: make(chan error),
			}

			errCh := make(chan error)
			go func() {
				<-helper.RouteDeactivateChan
			}()

			go func() {
				<-helper.RouteActivateChan
			}()
			go func() {
				rc.route.StartCh <- struct{}{}
			}()
			go func() {
				time.Sleep(3 * time.Second)
				cancel()
			}()

			go func() {
				errCh <- ProcessRoute(rc, helper, fgs)
			}()

			err := <-errCh
			convey.So(err, convey.ShouldBeNil)
		})
	})
}

func TestProcessRouteChannels(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		convey.Convey("Stop server stream", func() {
			ctx, cancel := context.WithCancel(context.Background())
			fgs := &FakeGRPCStream{
				Ctx: ctx,
			}
			helper := NewWebhookHelper()
			helper.ActiveEndpoints[rc.route.Webhook.Endpoint] = &Endpoint{
				DataCh: make(chan []byte),
			}
			helper.ActiveServers[rc.route.Webhook.Port] = &activeServer{
				errChan: make(chan error),
			}
			errCh := make(chan error)
			go func() {
				<-helper.RouteDeactivateChan
			}()
			go func() {
				errCh <- processChannels(rc, helper, fgs)
			}()
			cancel()
			err := <-errCh
			convey.So(err, convey.ShouldBeNil)
		})
		convey.Convey("Handle error", func() {
			fgs := &FakeGRPCStream{
				Ctx: context.Background(),
			}
			helper := NewWebhookHelper()
			helper.ActiveEndpoints[rc.route.Webhook.Endpoint] = &Endpoint{
				DataCh: make(chan []byte),
			}
			helper.ActiveServers[rc.route.Webhook.Port] = &activeServer{
				errChan: make(chan error),
			}
			errCh := make(chan error)
			err := fmt.Errorf("error")
			go func() {
				helper.ActiveServers[rc.route.Webhook.Port].errChan <- err
			}()
			go func() {
				errCh <- processChannels(rc, helper, fgs)
			}()
			newErr := <-errCh
			convey.So(newErr.Error(), convey.ShouldEqual, err.Error())
		})
	})
}

func TestFormatWebhookEndpoint(t *testing.T) {
	convey.Convey("Given a webhook endpoint, format it", t, func() {
		convey.So(FormatWebhookEndpoint("hello"), convey.ShouldEqual, "/hello")
	})
}

func TestValidateWebhook(t *testing.T) {
	convey.Convey("Given a webhook, validate it", t, func() {
		convey.So(ValidateWebhook(Hook), convey.ShouldBeNil)
	})
}

func TestGenerateFormattedURL(t *testing.T) {
	convey.Convey("Given a webhook, generate formatted URL", t, func() {
		convey.So(GenerateFormattedURL(Hook), convey.ShouldEqual, "test-url/fake")
	})
}

func TestNewWebhookHelper(t *testing.T) {
	convey.Convey("Make sure webhook helper is not empty", t, func() {
		helper := NewWebhookHelper()
		convey.So(helper, convey.ShouldNotBeNil)
	})
}
