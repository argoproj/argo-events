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

	"github.com/argoproj/argo-events/job"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cronlib "github.com/robfig/cron"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCalendarStartFailures(t *testing.T) {
	es := job.New(nil, nil, zap.NewNop())
	Calendar(es)
	calFactory, ok := es.GetCoreFactory(v1alpha1.SignalTypeCalendar)
	assert.True(t, ok, "calendar factory is not found")
	abstractSignal := job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Name: "nats-test",
			Calendar: &v1alpha1.CalendarSignal{
				Recurrence: []string{},
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}
	signal, err := calFactory.Create(abstractSignal)
	assert.Nil(t, err, "fail to create real calendar signal from abstract recurrence spec")

	testCh := make(chan job.Event)

	// test unknown signal
	err = signal.Start(testCh)
	assert.NotNil(t, err)

	// test invalid parsing of schedule
	abstractSignal = job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Name: "nats-test",
			Calendar: &v1alpha1.CalendarSignal{
				Schedule: "this is not a schedule",
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}
	signal, err = calFactory.Create(abstractSignal)
	assert.Nil(t, err, "failed to create real calendar signal from abstract schedule spec")
	err = signal.Start(testCh)
	assert.NotNil(t, err)

	// test invalid parsing of interval
	abstractSignal = job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Name: "nats-test",
			Calendar: &v1alpha1.CalendarSignal{
				Interval: "this is not a schedule",
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}
	signal, err = calFactory.Create(abstractSignal)
	assert.Nil(t, err)
	err = signal.Start(testCh)
	assert.NotNil(t, err)
}

func TestScheduleCalendar(t *testing.T) {
	es := job.New(nil, nil, zap.NewNop())
	Calendar(es)
	calFactory, ok := es.GetCoreFactory(v1alpha1.SignalTypeCalendar)
	assert.True(t, ok, "calendar factory is not found")
	abstractSignal := job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Name: "nats-test",
			Calendar: &v1alpha1.CalendarSignal{
				Schedule: "@every 1ms",
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}
	signal, err := calFactory.Create(abstractSignal)
	assert.Nil(t, err)
	testCh := make(chan job.Event)

	err = signal.Start(testCh)
	assert.Nil(t, err)

	time.Sleep(time.Millisecond)
	event, ok := <-testCh
	assert.True(t, ok)

	err = signal.Stop()
	assert.Nil(t, err)

	// ensure the event was correct
	assert.Equal(t, "", event.GetID())
	assert.Equal(t, "schedule: @every 1ms", event.GetSource())
	assert.Equal(t, signal, event.GetSignal())
	assert.True(t, time.Now().After(event.GetTimestamp()))
}

func TestIntervalCalendar(t *testing.T) {
	es := job.New(nil, nil, zap.NewNop())
	Calendar(es)
	calFactory, ok := es.GetCoreFactory(v1alpha1.SignalTypeCalendar)
	assert.True(t, ok, "calendar factory is not found")
	abstractSignal := job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Name: "nats-test",
			Calendar: &v1alpha1.CalendarSignal{
				Interval: "1ms",
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}
	signal, err := calFactory.Create(abstractSignal)
	assert.Nil(t, err)
	testCh := make(chan job.Event)

	err = signal.Start(testCh)
	assert.Nil(t, err)

	time.Sleep(time.Millisecond)
	event, ok := <-testCh
	assert.True(t, ok)

	err = signal.Stop()
	assert.Nil(t, err)

	// ensure the event was correct
	assert.Equal(t, "", event.GetID())
	assert.Equal(t, "interval: 1ms", event.GetSource())
	assert.Equal(t, signal, event.GetSignal())
	assert.True(t, time.Now().After(event.GetTimestamp()))
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

func TestGetUnknownSource(t *testing.T) {
	c := calendar{tickMethod: 3}
	assert.Equal(t, "unknown", c.getSource())
}
