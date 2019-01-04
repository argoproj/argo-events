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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	cronlib "github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

// StartEventSource starts an event source
func (ce *CalendarConfigExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ce.Log.Info().Str("event-source-name", *eventSource.Name).Msg("activating event source")
	cal, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}
	ce.Log.Info().Str("event-source-name", *eventSource.Name).Interface("config-value", *cal).Msg("calendar configuration")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ce.listenEvents(cal, eventSource, dataCh, errorCh, doneCh)

	return gateways.ConsumeEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ce.Log)
}

func resolveSchedule(cal *CalSchedule) (cronlib.Schedule, error) {
	if cal.Schedule != "" {
		// standard cron expression
		specParser := cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow)
		schedule, err := specParser.Parse(cal.Schedule)
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

// listenEvents fires an event when schedule is passed.
func (ce *CalendarConfigExecutor) listenEvents(cal *CalSchedule, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	schedule, err := resolveSchedule(cal)
	if err != nil {
		errorCh <- err
		return
	}

	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		errorCh <- err
		return
	}

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
	lastT := time.Now()

	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		ce.Log.Info().Str("event-source-name", *eventSource.Name).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				errorCh <- err
				return
			}
			dataCh <- payload
		case <-doneCh:
			return
		}
	}
}
