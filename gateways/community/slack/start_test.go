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

		rc.signingSecret = "9f3948e5883a385579c24f24492679b1"
		convey.Convey("Validate request signature", func() {
			writer := &gwcommon.FakeHttpWriter{}
			payload := []byte("payload=%7B%22type%22%3A%22block_actions%22%2C%22team%22%3A%7B%22id%22%3A%22T24QES0P2%22%2C%22domain%22%3A%22maersk-aa%22%7D%2C%22user%22%3A%7B%22id%22%3A%22U6X4142BW%22%2C%22username%22%3A%22adityasundaramurthy%22%2C%22name%22%3A%22adityasundaramurthy%22%2C%22team_id%22%3A%22T24QES0P2%22%7D%2C%22api_app_id%22%3A%22AKHHAA508%22%2C%22token%22%3A%22QxYPf6wsFFHWDanRC7dzkE6r%22%2C%22container%22%3A%7B%22type%22%3A%22message%22%2C%22message_ts%22%3A%221564564667.011700%22%2C%22channel_id%22%3A%22G3PM6F8FQ%22%2C%22is_ephemeral%22%3Afalse%7D%2C%22trigger_id%22%3A%22714362264135.72830884784.fa12b0cef241fdd10c03f5e30c0d5053%22%2C%22channel%22%3A%7B%22id%22%3A%22G3PM6F8FQ%22%2C%22name%22%3A%22privategroup%22%7D%2C%22message%22%3A%7B%22type%22%3A%22message%22%2C%22subtype%22%3A%22bot_message%22%2C%22text%22%3A%22Build+failed+for+platform-docs%5C%2Fmaster%22%2C%22ts%22%3A%221564564667.011700%22%2C%22username%22%3A%22Argo%22%2C%22bot_id%22%3A%22BKHFKMQVA%22%2C%22blocks%22%3A%5B%7B%22type%22%3A%22section%22%2C%22block_id%22%3A%22s0Ga%22%2C%22text%22%3A%7B%22type%22%3A%22mrkdwn%22%2C%22text%22%3A%22%3Ax%3A+%2ABuild+Failed%2A+for+%2Aplatform-docs%5C%2Fmaster%2A.+Started+by+%2ASinval+Vieira+Mendes+Neto%2A%22%2C%22verbatim%22%3Afalse%7D%7D%2C%7B%22type%22%3A%22actions%22%2C%22block_id%22%3A%22BhQAq%22%2C%22elements%22%3A%5B%7B%22type%22%3A%22button%22%2C%22action_id%22%3A%22l9lS%22%2C%22text%22%3A%7B%22type%22%3A%22plain_text%22%2C%22text%22%3A%22%3Abitbucket%3A+Commits%22%2C%22emoji%22%3Atrue%7D%2C%22url%22%3A%22https%3A%5C%2F%5C%2Fbitbucket.org%5C%2Fmaersk-analytics%5C%2Fplatform-docs%5C%2Fbranches%5C%2Fcompare%5C%2Fcbd11686494a7aee7f3a5298870102ad722231c6..1706a17e7a9a5c2610de03c79380a43975dcb21d%22%7D%2C%7B%22type%22%3A%22button%22%2C%22action_id%22%3A%223CyX%22%2C%22text%22%3A%7B%22type%22%3A%22plain_text%22%2C%22text%22%3A%22%3Aargo%3A+View+Build%22%2C%22emoji%22%3Atrue%7D%2C%22url%22%3A%22https%3A%5C%2F%5C%2Fargo-platform.maersk-digital.net%5C%2Fworkflows%5C%2Fargo-platform%5C%2Fplatform-docs--cbd11686--ci-1y1xo%22%7D%5D%7D%2C%7B%22type%22%3A%22divider%22%2C%22block_id%22%3A%22ktF%22%7D%5D%7D%2C%22response_url%22%3A%22https%3A%5C%2F%5C%2Fhooks.slack.com%5C%2Factions%5C%2FT24QES0P2%5C%2F714657463302%5C%2FwmNFMf3l9YSNfPt8DiNpXTUE%22%2C%22actions%22%3A%5B%7B%22action_id%22%3A%223CyX%22%2C%22block_id%22%3A%22BhQAq%22%2C%22text%22%3A%7B%22type%22%3A%22plain_text%22%2C%22text%22%3A%22%3Aargo%3A+View+Build%22%2C%22emoji%22%3Atrue%7D%2C%22type%22%3A%22button%22%2C%22action_ts%22%3A%221564619544.574857%22%7D%5D%7D")
			h := make(http.Header)
			h.Add("X-Slack-Signature", "v0=1de993bb52746615527d944f2bc12ceebf071e298ac1372a21987ea3aeabb962")
			h.Add("X-Slack-Request-Timestamp", "1564619544")
			rc.RouteHandler(writer, &http.Request{
				Body:   ioutil.NopCloser(bytes.NewReader(payload)),
				Header: h,
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
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
