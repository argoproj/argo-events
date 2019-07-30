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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	"github.com/nlopes/slack/slackevents"
	"github.com/smartystreets/goconvey/convey"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		rc := &RouteConfig{
			route:     gwcommon.GetFakeRoute(),
			clientset: fake.NewSimpleClientset(),
			namespace: "fake",
		}

		helper.ActiveEndpoints[rc.route.Webhook.Endpoint] = &gwcommon.Endpoint{
			DataCh: make(chan []byte),
		}

		convey.Convey("Inactive route should return 404", func() {
			writer := &gwcommon.FakeHttpWriter{}
			rc.RouteHandler(writer, &http.Request{})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		rc.token = "Jhj5dZrVaK7ZwHHjRyZWjbDl"
		helper.ActiveEndpoints[rc.route.Webhook.Endpoint].Active = true

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
			rc.RouteHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(payload)),
			})
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
				<-helper.ActiveEndpoints[rc.route.Webhook.Endpoint].DataCh
			}()

			rc.RouteHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewBuffer(payload)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusInternalServerError)
		})

		convey.Convey("Test an interaction action message", func() {
			writer := &gwcommon.FakeHttpWriter{}
			actionString := `{"type":"block_actions","team":{"id":"T9TK3CUKW","domain":"example"},"user":{"id":"UA8RXUSPL","username":"jtorrance","team_id":"T9TK3CUKW"},"api_app_id":"AABA1ABCD","token":"9s8d9as89d8as9d8as989","container":{"type":"message_attachment","message_ts":"1548261231.000200","attachment_id":1,"channel_id":"CBR2V3XEX","is_ephemeral":false,"is_app_unfurl":false},"trigger_id":"12321423423.333649436676.d8c1bb837935619ccad0f624c448ffb3","channel":{"id":"CBR2V3XEX","name":"review-updates"},"message":{"bot_id":"BAH5CA16Z","type":"message","text":"This content can't be displayed.","user":"UAJ2RU415","ts":"1548261231.000200"},"response_url":"https://hooks.slack.com/actions/AABA1ABCD/1232321423432/D09sSasdasdAS9091209","actions":[{"action_id":"WaXA","block_id":"=qXel","text":{"type":"plain_text","text":"View","emoji":true},"value":"click_me_123","type":"button","action_ts":"1548426417.840180"}]}`
			payload := []byte(`payload=` + actionString)
			out := make(chan []byte)
			go func() {
				out <- <-helper.ActiveEndpoints[rc.route.Webhook.Endpoint].DataCh
			}()

			var buf bytes.Buffer
			buf.Write(payload)

			headers := make(map[string][]string)
			headers["Content-Type"] = append(headers["Content-Type"], "application/x-www-form-urlencoded")
			rc.RouteHandler(writer, &http.Request{
				Method: http.MethodPost,
				Header: headers,
				Body:   ioutil.NopCloser(strings.NewReader(buf.String())),
			})
			result := <-out
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			convey.So(string(result), convey.ShouldContainSubstring, "\"type\":\"block_actions\"")
			convey.So(string(result), convey.ShouldContainSubstring, "\"token\":\"9s8d9as89d8as9d8as989\"")
		})
	})
}
