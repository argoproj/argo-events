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

package slack

import (
	"bytes"
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/nlopes/slack/slackevents"
	"io/ioutil"
	"net/http"
	"testing"

	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/smartystreets/goconvey/convey"
)

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		rc := gwcommon.GetFakeRouteConfig()

		helper.ActiveEndpoints[rc.Webhook.Endpoint] = &gwcommon.Endpoint{
			DataCh: make(chan []byte),
		}

		convey.Convey("Inactive route should return 404", func() {
			writer := &gwcommon.FakeHttpWriter{}
			RouteActiveHandler(writer, &http.Request{}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		rc.Configs[labelSlackToken] = "Jhj5dZrVaK7ZwHHjRyZWjbDl"
		helper.ActiveEndpoints[rc.Webhook.Endpoint].Active = true

		convey.Convey("Test url verification request", func() {
			writer := &gwcommon.FakeHttpWriter{}
			urlVer := slackevents.EventsAPIURLVerificationEvent{
				Type:      slackevents.URLVerification,
				Token:     "Jhj5dZrVaK7ZwHHjRyZWjbDl",
				Challenge: "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P",
			}
			payload, err := yaml.Marshal(urlVer)
			convey.So(err, convey.ShouldBeNil)
			convey.So(payload, convey.ShouldNotBeNil)
			RouteActiveHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(payload)),
			}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusInternalServerError)
		})

		convey.Convey("Test an event notification", func() {
			writer := &gwcommon.FakeHttpWriter{}
			event := []byte(`
{
"type": "name_of_event",
"event_ts": "1234567890.123456",
"user": "UXXXXXXX1"
}
`)

			var j json.RawMessage
			j = event
			ce := slackevents.EventsAPICallbackEvent{
				Token:     "Jhj5dZrVaK7ZwHHjRyZWjbDl",
				Type:      slackevents.CallbackEvent,
				EventTime: 1234567890,
				APIAppID:  "AXXXXXXXXX",
				AuthedUsers: []string{
					"UXXXXXXX1",
					"UXXXXXXX2",
				},
				EventID:    "Ev08MFMKH6",
				InnerEvent: &j,
			}
			payload, err := yaml.Marshal(ce)
			convey.So(err, convey.ShouldBeNil)

			go func() {
				<-helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh
			}()

			RouteActiveHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewBuffer(payload)),
			}, rc)
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusInternalServerError)
		})

	})
}
