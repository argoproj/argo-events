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

package calendar

import (
	"encoding/json"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
)

func TestResolveSchedule(t *testing.T) {
	convey.Convey("Given a calendar schedule, resolve it", t, func() {
		schedule, err := resolveSchedule(&v1alpha1.CalendarEventSource{
			Schedule: "* * * * *",
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(schedule, convey.ShouldNotBeNil)
	})
}

func TestListenEvents(t *testing.T) {
	convey.Convey("Given a calendar schedule, listen events", t, func() {
		listener := &EventListener{
			Logger: common.NewArgoEventsLogger(),
		}

		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{}, 1)
		dataCh2 := make(chan []byte)

		go func() {
			data := <-dataCh
			dataCh2 <- data
		}()

		payload := []byte(`"{\r\n\"hello\": \"world\"\r\n}"`)
		raw := json.RawMessage(payload)

		calendarEventSource := &v1alpha1.CalendarEventSource{
			Schedule:    "* * * * *",
			UserPayload: &raw,
		}

		body, err := yaml.Marshal(calendarEventSource)
		convey.So(err, convey.ShouldBeNil)

		go listener.listenEvents(&gateways.EventSource{
			Name:  "fake",
			Value: body,
			Id:    "1234",
			Type:  string(apicommon.CalendarEvent),
		}, dataCh, errorCh, doneCh)

		data := <-dataCh2
		doneCh <- struct{}{}

		var cal *response
		err = yaml.Unmarshal(data, &cal)
		convey.So(err, convey.ShouldBeNil)

		payload, err = cal.UserPayload.MarshalJSON()
		convey.So(err, convey.ShouldBeNil)

		convey.So(string(payload), convey.ShouldEqual, `"{\r\n\"hello\": \"world\"\r\n}"`)
	})
}
