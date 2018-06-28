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

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/shared"
	cronlib "github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EventType = "com.github.argoproj.calendar"
)

const (
	tickMethodSchedule = iota
	tickMethodInterval
)

type calendar struct {
	tickMethod     int
	schedule       cronlib.Schedule
	exclusionDates []time.Time
	stop           chan struct{}
}

// New creates a new calendar signaler
func New() shared.Signaler {
	return &calendar{
		stop: make(chan struct{}),
	}
}

func (c *calendar) Start(signal *v1alpha1.Signal) (<-chan *v1alpha1.Event, error) {
	// parse out the calendar configurations
	var err error
	if signal.Calendar.Schedule != "" {
		c.tickMethod = tickMethodSchedule
		c.schedule, err = cronlib.Parse(signal.Calendar.Schedule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schedule %s from calendar signal. Cause: %+v", signal.Calendar.Schedule, err.Error())
		}
	} else if signal.Calendar.Interval != "" {
		c.tickMethod = tickMethodInterval
		intervalDuration, err := time.ParseDuration(signal.Calendar.Interval)
		if err != nil {
			return nil, fmt.Errorf("failed to parse interval %s from calendar signal. Cause: %+v", signal.Calendar.Interval, err.Error())
		}
		c.schedule = cronlib.ConstantDelaySchedule{Delay: intervalDuration}
	} else {
		return nil, fmt.Errorf("calendar signal must contain either a schedule or interval")
	}

	c.exclusionDates = parseExclusionDates(signal.Calendar.Recurrence)

	events := make(chan *v1alpha1.Event)
	go c.handleEvents(events)
	return events, nil
}

func (c *calendar) Stop() error {
	c.stop <- struct{}{}
	close(c.stop)
	return nil
}

func (c *calendar) handleEvents(events chan *v1alpha1.Event) {
	defer close(events)
	eventTimer := c.getEventTimer()
	for t := range eventTimer {
		event := &v1alpha1.Event{
			Context: v1alpha1.EventContext{
				EventID:            t.String(),
				EventType:          EventType,
				CloudEventsVersion: shared.CloudEventsVersion,
				EventTime:          metav1.Time{Time: t},
				Extensions:         make(map[string]string),
			},
		}
		events <- event
	}
}

func (c *calendar) getEventTimer() <-chan time.Time {
	lastT := time.Now()
	eventTimer := make(chan time.Time)
	go func() {
		defer close(eventTimer)
		for {
			t := c.getNextTime(lastT)
			timer := time.After(time.Until(t))
			log.Printf("expected next calendar event %s", t)
			select {
			case tx := <-timer:
				eventTimer <- tx
				lastT = tx
			case <-c.stop:
				return
			}
		}
	}()
	return eventTimer
}

func (c *calendar) getNextTime(lastEventTime time.Time) time.Time {
	nextEventTime := c.schedule.Next(lastEventTime)
	nextYear := nextEventTime.Year()
	nextMonth := nextEventTime.Month()
	nextDay := nextEventTime.Day()
	for _, exDate := range c.exclusionDates {
		// if exDate == nextEvent, then we need to skip this and get the next
		if exDate.Year() == nextYear && exDate.Month() == nextMonth && exDate.Day() == nextDay {
			return c.getNextTime(nextEventTime)
		}
	}
	return nextEventTime
}
