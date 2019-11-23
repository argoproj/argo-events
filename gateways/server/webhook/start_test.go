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
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/smartystreets/goconvey/convey"
)

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		router := &Router{
			route: webhook.GetFakeRoute(),
		}
		route := router.route
		route.Context.Method = http.MethodGet

		writer := &webhook.FakeHttpWriter{}

		convey.Convey("Inactive route should return error", func() {
			router.HandleRoute(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader([]byte("hello"))),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		route.Active = true

		convey.Convey("Active route with correct method should return success", func() {
			dataCh := make(chan []byte)
			go func() {
				resp := <-route.DataCh
				dataCh <- resp
			}()

			router.HandleRoute(writer, &http.Request{
				Body:   ioutil.NopCloser(bytes.NewReader([]byte("fake notification"))),
				Method: http.MethodGet,
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			data := <-dataCh
			convey.So(string(data), convey.ShouldEqual, "fake notification")
		})

		convey.Convey("Active route with incorrect method should return failure", func() {
			router.HandleRoute(writer, &http.Request{
				Body:   ioutil.NopCloser(bytes.NewReader([]byte("fake notification"))),
				Method: http.MethodHead,
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})
	})
}
