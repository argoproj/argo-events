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
	"github.com/argoproj/argo-events/gateways/server"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestResolveSchedule(t *testing.T) {
	schedule, err := resolveSchedule(&v1alpha1.CalendarEventSource{
		Schedule: "* * * * *",
	})
	assert.Nil(t, err)
	assert.NotNil(t, schedule)
}

func TestListenEvents(t *testing.T) {
	listener := &EventListener{
		Logger: common.NewArgoEventsLogger(),
	}
	payload := []byte(`"{\r\n\"hello\": \"world\"\r\n}"`)
	raw := json.RawMessage(payload)

	calendarEventSource := &v1alpha1.CalendarEventSource{
		Interval:    "2s",
		UserPayload: &raw,
	}

	body, err := yaml.Marshal(calendarEventSource)
	assert.Nil(t, err)

	channels := &server.Channels{
		Data: make(chan []byte),
		Stop: make(chan struct{}),
		Done: make(chan struct{}),
	}
	go func() {
		data := <-channels.Data
		var cal *events.CalendarEventData
		err = yaml.Unmarshal(data, &cal)
		assert.Nil(t, err)

		payload, err = cal.UserPayload.MarshalJSON()
		assert.Nil(t, err)

		assert.Equal(t, `"{\r\n\"hello\": \"world\"\r\n}"`, string(payload))
		channels.Done <- struct{}{}
	}()

	err = listener.listenEvents(&gateways.EventSource{
		Name:  "fake",
		Value: body,
		Id:    "1234",
		Type:  string(apicommon.CalendarEvent),
	}, channels)
	assert.Nil(t, err)
}
