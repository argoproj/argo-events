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
	"fmt"
	"log"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	cronlib "github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EventType = "com.github.argoproj.calendar"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
type calendar struct{}

// New creates a new calendar listener
func New() sdk.Listener {
	return new(calendar)
}

func (c *calendar) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	schedule, err := resolveSchedule(signal.Calendar)
	if err != nil {
		return nil, err
	}
	exDates, err := common.ParseExclusionDates(signal.Calendar.Recurrence)
	if err != nil {
		return nil, err
	}

	events := make(chan *v1alpha1.Event)

	var next Next
	next = func(last time.Time) time.Time {
		nextT := schedule.Next(last)
		nextYear := nextT.Year()
		nextMonth := nextT.Month()
		nextDay := nextT.Day()
		for _, exDate := range exDates {
			// if exDate == nextEvent, then we need to skip this and get the next
			if exDate.Year() == nextYear && exDate.Month() == nextMonth && exDate.Day() == nextDay {
				return next(nextT)
			}
		}
		return nextT
	}

	// start handling events
	go c.handleEvents(events, next, done)
	return events, nil
}

func (c *calendar) handleEvents(events chan *v1alpha1.Event, next Next, done <-chan struct{}) {
	defer close(events)
	eventTimer := c.getEventTimer(next, done)
	for t := range eventTimer {
		event := &v1alpha1.Event{
			Context: v1alpha1.EventContext{
				EventID:            t.String(),
				EventType:          EventType,
				CloudEventsVersion: sdk.CloudEventsVersion,
				EventTime:          metav1.Time{Time: t},
				Extensions:         make(map[string]string),
			},
		}
		events <- event
	}
}

func (c *calendar) getEventTimer(next Next, done <-chan struct{}) <-chan time.Time {
	lastT := time.Now()
	eventTimer := make(chan time.Time)
	go func() {
		defer close(eventTimer)
		for {
			t := next(lastT)
			timer := time.After(time.Until(t))
			log.Printf("expected next calendar event %s", t)
			select {
			case tx := <-timer:
				eventTimer <- tx
				lastT = tx
			case <-done:
				return
			}
		}
	}()
	return eventTimer
}

func resolveSchedule(cal *v1alpha1.CalendarSignal) (cronlib.Schedule, error) {
	if cal.Schedule != "" {
		schedule, err := cronlib.Parse(cal.Schedule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schedule %s from calendar signal. Cause: %+v", cal.Schedule, err.Error())
		}
		return schedule, nil
	} else if cal.Interval != "" {
		intervalDuration, err := time.ParseDuration(cal.Interval)
		if err != nil {
			return nil, fmt.Errorf("failed to parse interval %s from calendar signal. Cause: %+v", cal.Interval, err.Error())
		}
		schedule := cronlib.ConstantDelaySchedule{Delay: intervalDuration}
		return schedule, nil
	} else {
		return nil, fmt.Errorf("calendar signal must contain either a schedule or interval")
	}
}
