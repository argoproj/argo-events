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
	"testing"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cronlib "github.com/robfig/cron"
	"github.com/stretchr/testify/assert"
)

func TestCalendarStartFailures(t *testing.T) {
	signal := v1alpha1.Signal{
		Name: "nats-test",
		Calendar: &v1alpha1.CalendarSignal{
			Recurrence: []string{},
		},
	}
	cal := New()

	// test unknown signal
	_, err := cal.Start(&signal)
	if err == nil {
		t.Errorf("expected a non nil error for an unknown calendar signal")
	}

	// test invalid parsing of schedule
	signal = v1alpha1.Signal{
		Name: "nats-test",
		Calendar: &v1alpha1.CalendarSignal{
			Schedule: "this is not a schedule",
		},
	}
	_, err = cal.Start(&signal)
	if err == nil {
		t.Errorf("expected a non nil error for invalid parsing of schedule")
	}

	// test invalid parsing of interval
	signal = v1alpha1.Signal{
		Name: "nats-test",
		Calendar: &v1alpha1.CalendarSignal{
			Interval: "this is not a schedule",
		},
	}
	_, err = cal.Start(&signal)
	if err == nil {
		t.Errorf("expected a non nil error for invalid parsing of interval")
	}
}

func TestScheduleCalendar(t *testing.T) {
	cal := New()
	signal := v1alpha1.Signal{
		Name: "nats-test",
		Calendar: &v1alpha1.CalendarSignal{
			Schedule: "@every 1ms",
		},
	}

	events, err := cal.Start(&signal)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Millisecond)
	event, ok := <-events
	if !ok {
		t.Errorf("expected an event but found none")
	}

	err = cal.Stop()
	if err != nil {
		t.Error(err)
	}

	// ensure the event was correct
	if event.Context.EventType != EventType {
		t.Errorf("event context EventType\nexpected: %s\nactual: %s", EventType, event.Context.EventType)
	}
}

func TestIntervalCalendar(t *testing.T) {
	cal := New()
	signal := v1alpha1.Signal{
		Name: "nats-test",
		Calendar: &v1alpha1.CalendarSignal{
			Interval: "1ms",
		},
	}

	events, err := cal.Start(&signal)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Millisecond)
	event, ok := <-events
	if !ok {
		t.Errorf("expected an event but found none")
	}

	err = cal.Stop()
	if err != nil {
		t.Error(err)
	}

	// ensure the event was correct
	// ensure the event was correct
	if event.Context.EventType != EventType {
		t.Errorf("event context EventType\nexpected: %s\nactual: %s", EventType, event.Context.EventType)
	}
}

func TestGetNextTime(t *testing.T) {
	c := calendar{
		schedule:       cronlib.ConstantDelaySchedule{Delay: time.Minute},
		exclusionDates: []time.Time{time.Date(2018, time.May, 10, 0, 0, 0, 0, time.UTC)},
	}
	lastEventTime := time.Date(2018, time.May, 9, 23, 59, 0, 0, time.UTC)
	nextEventTime := c.getNextTime(lastEventTime)
	expectedNextEventTime := time.Date(2018, time.May, 11, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedNextEventTime, nextEventTime)

	lastEventTime = time.Date(2018, time.May, 9, 23, 58, 59, 0, time.UTC)
	nextEventTime = c.getNextTime(lastEventTime)
	expectedNextEventTime = time.Date(2018, time.May, 9, 23, 59, 59, 0, time.UTC)
	assert.Equal(t, expectedNextEventTime, nextEventTime)
}
