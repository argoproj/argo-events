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
	"fmt"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	cronlib "github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

// EventSourceListener implements Eventing for calendar based events
type EventSourceListener struct {
	Logger *logrus.Logger
}

// response is the event payload that is sent as response to sensor
type response struct {
	// EventTime is time at which event occurred
	EventTime time.Time `json:"eventTime"`
	// UserPayload if any
	UserPayload *json.RawMessage `json:"userPayload"`
}

// Next is a function to compute the next event time from a given time
type Next func(time.Time) time.Time

// StartEventSource starts an event source
func (listener *EventSourceListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	log := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("activating event source")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents fires an event when schedule is passed.
func (listener *EventSourceListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	var calendarEventSource *v1alpha1.CalendarEventSource
	if err := yaml.Unmarshal(eventSource.Value, &calendarEventSource); err != nil {
		errorCh <- err
		return
	}

	schedule, err := resolveSchedule(calendarEventSource)
	if err != nil {
		errorCh <- err
		return
	}

	exDates, err := common.ParseExclusionDates(calendarEventSource.ExclusionDates)
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
	var location *time.Location
	if calendarEventSource.Timezone != "" {
		location, err = time.LoadLocation(calendarEventSource.Timezone)
		if err != nil {
			errorCh <- err
			return
		}
		lastT = lastT.In(location)
	}

	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		listener.Logger.WithFields(
			map[string]interface{}{
				common.LabelEventSource: eventSource.Name,
				common.LabelTime:        t.UTC().String(),
			}).Info("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			if location != nil {
				lastT = lastT.In(location)
			}
			response := &response{
				EventTime:   tx,
				UserPayload: calendarEventSource.UserPayload,
			}
			payload, err := json.Marshal(response)
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

func resolveSchedule(cal *v1alpha1.CalendarEventSource) (cronlib.Schedule, error) {
	if cal.Schedule != "" {
		// standard cron expression
		specParser := cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow)
		schedule, err := specParser.Parse(cal.Schedule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schedule %s from calendar event. Cause: %+v", cal.Schedule, err.Error())
		}
		return schedule, nil
	} else if cal.Interval != "" {
		intervalDuration, err := time.ParseDuration(cal.Interval)
		if err != nil {
			return nil, fmt.Errorf("failed to parse interval %s from calendar event. Cause: %+v", cal.Interval, err.Error())
		}
		schedule := cronlib.ConstantDelaySchedule{Delay: intervalDuration}
		return schedule, nil
	} else {
		return nil, fmt.Errorf("calendar event must contain either a schedule or interval")
	}
}
