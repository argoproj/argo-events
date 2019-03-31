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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestResolveSchedule(t *testing.T) {
	convey.Convey("Given a calendar schedule, resolve it", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ps, convey.ShouldNotBeNil)

		schedule, err := resolveSchedule(ps.(*calSchedule))
		convey.So(err, convey.ShouldBeNil)
		convey.So(schedule, convey.ShouldNotBeNil)
	})
}

func TestListenEvents(t *testing.T) {
	convey.Convey("Given a calendar schedule, listen events", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ps, convey.ShouldNotBeNil)

		ese := &CalendarEventSourceExecutor{
			Log: common.NewArgoEventsLogger(),
		}
		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{}, 1)
		dataCh2 := make(chan []byte)

		go func() {
			data := <-dataCh
			dataCh2 <- data
		}()

		go ese.listenEvents(ps.(*calSchedule), &gateways.EventSource{
			Name: "fake",
			Data: es,
			Id:   "1234",
		}, dataCh, errorCh, doneCh)

		data := <-dataCh2
		doneCh <- struct{}{}

		var cal *calResponse
		err = yaml.Unmarshal(data, &cal)
		convey.So(err, convey.ShouldBeNil)

		convey.So(cal.UserPayload, convey.ShouldEqual, "{\r\n\"hello\": \"world\"\r\n}")
	})
}
