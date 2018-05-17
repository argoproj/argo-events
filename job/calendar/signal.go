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
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/blackrock/axis/job"
	cronlib "github.com/robfig/cron"
)

const (
	tickMethodSchedule = iota
	tickMethodInterval
)

type calendar struct {
	job.AbstractSignal
	tickMethod     int
	schedule       cronlib.Schedule
	exclusionDates []time.Time
	stop           chan struct{}
	wg             sync.WaitGroup
}

func (c *calendar) Start(events chan job.Event) error {
	// parse out the calendar configurations
	var err error
	if c.AbstractSignal.Calendar.Schedule != "" {
		c.tickMethod = tickMethodSchedule
		c.schedule, err = cronlib.Parse(c.AbstractSignal.Calendar.Schedule)
		if err != nil {
			return fmt.Errorf("failed to parse schedule %s from calendar signal. Cause: %+v", c.AbstractSignal.Calendar.Schedule, err.Error())
		}
	} else if c.AbstractSignal.Calendar.Interval != "" {
		c.tickMethod = tickMethodInterval
		intervalDuration, err := time.ParseDuration(c.AbstractSignal.Calendar.Interval)
		if err != nil {
			return fmt.Errorf("failed to parse interval %s from calendar signal. Cause: %+v", c.AbstractSignal.Calendar.Interval, err.Error())
		}
		c.schedule = cronlib.ConstantDelaySchedule{Delay: intervalDuration}
	} else {
		return fmt.Errorf("calendar signal must contain either a schedule or interval")
	}

	c.exclusionDates = parseExclusionDates(c.Calendar.Recurrence)

	c.wg.Add(1)
	go c.handleEvents(events)
	return nil
}

func (c *calendar) Stop() error {
	c.stop <- struct{}{}
	close(c.stop)
	c.wg.Wait()
	return nil
}

func (c *calendar) handleEvents(events chan job.Event) {
	eventTimer := c.getEventTimer()
	for t := range eventTimer {
		event := &event{
			calendar:  c,
			timestamp: t,
		}
		// perform constraint checks
		err := c.CheckConstraints(event.GetTimestamp())
		if err != nil {
			event.SetError(err)
		}
		c.Log.Debug("sending calendar event", zap.String("nodeID", event.GetID()))
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
			c.Log.Debug("expected next calendar event", zap.Time("t", t))
			select {
			case tx := <-timer:
				eventTimer <- tx
				lastT = tx
			case <-c.stop:
				c.wg.Done()
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

func (c *calendar) getSource() string {
	switch c.tickMethod {
	case tickMethodSchedule:
		return "schedule: " + c.AbstractSignal.Calendar.Schedule
	case tickMethodInterval:
		return "interval: " + c.AbstractSignal.Calendar.Interval
	default:
		return "unknown"
	}
}
