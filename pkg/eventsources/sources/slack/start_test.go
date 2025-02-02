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

package slack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/slack-go/slack/slackevents"
	"github.com/smartystreets/goconvey/convey"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
)

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		router := &Router{
			route:            webhook.GetFakeRoute(),
			slackEventSource: &v1alpha1.SlackEventSource{},
		}

		convey.Convey("Inactive route should return 404", func() {
			writer := &webhook.FakeHttpWriter{}
			router.HandleRoute(writer, &http.Request{})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		router.token = "Jhj5dZrVaK7ZwHHjRyZWjbDl"
		router.route.Active = true

		convey.Convey("Test url verification request", func() {
			writer := &webhook.FakeHttpWriter{}
			urlVer := slackevents.EventsAPIURLVerificationEvent{
				Type:      slackevents.URLVerification,
				Token:     "Jhj5dZrVaK7ZwHHjRyZWjbDl",
				Challenge: "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P",
			}
			payload, err := yaml.Marshal(urlVer)
			convey.So(err, convey.ShouldBeNil)
			convey.So(payload, convey.ShouldNotBeNil)
			router.HandleRoute(writer, &http.Request{
				Body: io.NopCloser(bytes.NewReader(payload)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestSlackSignature(t *testing.T) {
	convey.Convey("Given a route that receives a message from Slack", t, func() {
		router := &Router{
			route:            webhook.GetFakeRoute(),
			slackEventSource: &v1alpha1.SlackEventSource{},
		}

		router.signingSecret = "abcdefghiklm1234567890"
		convey.Convey("Validate request signature", func() {
			writer := &webhook.FakeHttpWriter{}
			payload := []byte("payload=%7B%22type%22%3A%22block_actions%22%2C%22team%22%3A%7B%22id%22%3A%22T0CAG%22%2C%22domain%22%3A%22acme-creamery%22%7D%2C%22user%22%3A%7B%22id%22%3A%22U0CA5%22%2C%22username%22%3A%22Amy%20McGee%22%2C%22name%22%3A%22Amy%20McGee%22%2C%22team_id%22%3A%22T3MDE%22%7D%2C%22api_app_id%22%3A%22A0CA5%22%2C%22token%22%3A%22Shh_its_a_seekrit%22%2C%22container%22%3A%7B%22type%22%3A%22message%22%2C%22text%22%3A%22The%20contents%20of%20the%20original%20message%20where%20the%20action%20originated%22%7D%2C%22trigger_id%22%3A%2212466734323.1395872398%22%2C%22response_url%22%3A%22https%3A%2F%2Fwww.postresponsestome.com%2FT123567%2F1509734234%22%2C%22actions%22%3A%5B%7B%22type%22%3A%22button%22%2C%22block_id%22%3A%22actionblock789%22%2C%22action_id%22%3A%2227S%22%2C%22text%22%3A%7B%22type%22%3A%22plain_text%22%2C%22text%22%3A%22Link%20Button%22%2C%22emoji%22%3Atrue%7D%2C%22action_ts%22%3A%221564701248.149432%22%7D%5D%7D")
			h := make(http.Header)

			rts := int(time.Now().UTC().UnixNano())
			hmac := hmac.New(sha256.New, []byte(router.signingSecret))
			b := strings.Join([]string{"v0", strconv.Itoa(rts), string(payload)}, ":")
			_, err := hmac.Write([]byte(b))
			convey.So(err, convey.ShouldBeNil)
			hash := hex.EncodeToString(hmac.Sum(nil))
			genSig := strings.TrimRight(strings.Join([]string{"v0=", hash}, ""), "\n")
			h.Add("Content-Type", "application/x-www-form-urlencoded")
			h.Add("X-Slack-Signature", genSig)
			h.Add("X-Slack-Request-Timestamp", strconv.FormatInt(int64(rts), 10))

			router.route.Active = true

			go func() {
				temp := <-router.route.DispatchChan
				temp.SuccessChan <- true
			}()

			router.HandleRoute(writer, &http.Request{
				Body:   io.NopCloser(bytes.NewReader(payload)),
				Header: h,
				Method: "POST",
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
		})
	})
}

func TestInteractionHandler(t *testing.T) {
	convey.Convey("Given a route that receives an interaction event", t, func() {
		router := &Router{
			route:            webhook.GetFakeRoute(),
			slackEventSource: &v1alpha1.SlackEventSource{},
		}

		convey.Convey("Test an interaction action message", func() {
			writer := &webhook.FakeHttpWriter{}
			actionString := `{"type":"block_actions","team":{"id":"T9TK3CUKW","domain":"example"},"user":{"id":"UA8RXUSPL","username":"jtorrance","team_id":"T9TK3CUKW"},"api_app_id":"AABA1ABCD","token":"9s8d9as89d8as9d8as989","container":{"type":"message_attachment","message_ts":"1548261231.000200","attachment_id":1,"channel_id":"CBR2V3XEX","is_ephemeral":false,"is_app_unfurl":false},"trigger_id":"12321423423.333649436676.d8c1bb837935619ccad0f624c448ffb3","channel":{"id":"CBR2V3XEX","name":"review-updates"},"message":{"bot_id":"BAH5CA16Z","type":"message","text":"This content can't be displayed.","user":"UAJ2RU415","ts":"1548261231.000200"},"response_url":"https://hooks.slack.com/actions/AABA1ABCD/1232321423432/D09sSasdasdAS9091209","actions":[{"action_id":"WaXA","block_id":"=qXel","text":{"type":"plain_text","text":"View","emoji":true},"value":"click_me_123","type":"button","action_ts":"1548426417.840180"}]}`
			payload := []byte(`payload=` + actionString)
			out := make(chan *webhook.Dispatch)
			router.route.Active = true

			go func() {
				temp := <-router.route.DispatchChan
				temp.SuccessChan <- true
				out <- temp
			}()

			var buf bytes.Buffer
			buf.Write(payload)

			headers := make(map[string][]string)
			headers["Content-Type"] = append(headers["Content-Type"], "application/x-www-form-urlencoded")
			router.HandleRoute(writer, &http.Request{
				Method: http.MethodPost,
				Header: headers,
				Body:   io.NopCloser(strings.NewReader(buf.String())),
			})
			result := <-out
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			convey.So(string(result.Data), convey.ShouldContainSubstring, "\"type\":\"block_actions\"")
			convey.So(string(result.Data), convey.ShouldContainSubstring, "\"token\":\"9s8d9as89d8as9d8as989\"")
		})
	})
}

func TestSlackCommandHandler(t *testing.T) {
	convey.Convey("Given a route that receives a slash command event", t, func() {
		router := &Router{
			route:            webhook.GetFakeRoute(),
			slackEventSource: &v1alpha1.SlackEventSource{},
		}

		convey.Convey("Test a slash command message", func() {
			writer := &webhook.FakeHttpWriter{}
			// Test values pulled from example here: https://api.slack.com/interactivity/slash-commands#app_command_handling
			payload := []byte(`token=gIkuvaNzQIHg97ATvDxqgjtO&team_id=T0001&team_domain=example&enterprise_id=E0001&enterprise_name=Globular%20Construct%20Inc&channel_id=C2147483705&channel_name=test&user_id=U2147483697&user_name=Steve&command=/weather&text=94070&response_url=https://hooks.slack.com/commands/1234/5678&trigger_id=13345224609.738474920.8088930838d88f008e0&api_app_id=A123456`)
			out := make(chan *webhook.Dispatch)
			router.route.Active = true

			go func() {
				temp := <-router.route.DispatchChan
				temp.SuccessChan <- true
				out <- temp
			}()

			var buf bytes.Buffer
			buf.Write(payload)

			headers := make(map[string][]string)
			headers["Content-Type"] = append(headers["Content-Type"], "application/x-www-form-urlencoded")
			router.HandleRoute(writer, &http.Request{
				Method: http.MethodPost,
				Header: headers,
				Body:   io.NopCloser(strings.NewReader(buf.String())),
			})
			result := <-out
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
			convey.So(string(result.Data), convey.ShouldContainSubstring, "\"command\":\"/weather\"")
			convey.So(string(result.Data), convey.ShouldContainSubstring, "\"token\":\"gIkuvaNzQIHg97ATvDxqgjtO\"")
		})
	})
}

func TestEventHandler(t *testing.T) {
	convey.Convey("Given a route that receives an event", t, func() {
		router := &Router{
			route:            webhook.GetFakeRoute(),
			slackEventSource: &v1alpha1.SlackEventSource{},
		}

		convey.Convey("Test an event notification", func() {
			writer := &webhook.FakeHttpWriter{}
			event := []byte(`
{
"type": "name_of_event",
"event_ts": "1234567890.123456",
"user": "UXXXXXXX1"
}
`)

			j := json.RawMessage(event)
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

			router.route.Active = true

			go func() {
				<-router.route.DispatchChan
			}()

			router.HandleRoute(writer, &http.Request{
				Body: io.NopCloser(bytes.NewBuffer(payload)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusInternalServerError)
		})
	})
}
