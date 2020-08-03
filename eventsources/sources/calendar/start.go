/*
Copyright 2020 BlackRock, Inc.

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
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	cronlib "github.com/robfig/cron"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for calendar based events
type EventListener struct {
	EventSourceName     string
	EventName           string
	CalendarEventSource v1alpha1.CalendarEventSource
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.CalendarEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName()).Desugar()
	log.Info("started processing the calendar event source...")

	calendarEventSource := &el.CalendarEventSource
	log.Info("resolving calendar schedule...")
	schedule, err := resolveSchedule(calendarEventSource)
	if err != nil {
		return err
	}

	log.Info("parsing exclusion dates if any...")
	exDates, err := common.ParseExclusionDates(calendarEventSource.ExclusionDates)
	if err != nil {
		return err
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
		log.Info("loading location for the schedule...", zap.Any("location", calendarEventSource.Timezone))
		location, err = time.LoadLocation(calendarEventSource.Timezone)
		if err != nil {
			return errors.Wrapf(err, "failed to load location for event source %s / %s", el.GetEventSourceName(), el.GetEventName())
		}
		lastT = lastT.In(location)
	}

	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		log.Info("expected next calendar event", zap.Any(logging.LabelTime, t.UTC().String()))
		select {
		case tx := <-timer:
			lastT = tx
			if location != nil {
				lastT = lastT.In(location)
			}
			response := &events.CalendarEventData{
				EventTime:   tx.String(),
				UserPayload: calendarEventSource.UserPayload,
				Metadata:    calendarEventSource.Metadata,
			}
			payload, err := json.Marshal(response)
			if err != nil {
				log.Error("failed to marshal the event data", zap.Error(err))
				// no need to continue as further event payloads will suffer same fate as this one.
				return errors.Wrapf(err, "failed to marshal the event data for event source %s / %s", el.GetEventSourceName(), el.GetEventName())
			}
			log.Info("dispatching calendar event...")
			err = dispatch(payload)
			if err != nil {
				log.Error("failed to dispatch calendar event", zap.Error(err))
			}
		case <-ctx.Done():
			log.Info("exiting calendar event listener...")
			return nil
		}
	}
}

// Next is a function to compute the next event time from a given time
type Next func(time.Time) time.Time

// resolveSchedule parses the schedule and returns a valid cron schedule
func resolveSchedule(cal *v1alpha1.CalendarEventSource) (cronlib.Schedule, error) {
	if cal.Schedule != "" {
		// standard cron expression
		specParser := cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow)
		schedule, err := specParser.Parse(cal.Schedule)
		if err != nil {
			return nil, errors.Errorf("failed to parse schedule %s from calendar event. Cause: %+v", cal.Schedule, err.Error())
		}
		return schedule, nil
	}
	if cal.Interval != "" {
		intervalDuration, err := time.ParseDuration(cal.Interval)
		if err != nil {
			return nil, errors.Errorf("failed to parse interval %s from calendar event. Cause: %+v", cal.Interval, err.Error())
		}
		schedule := cronlib.ConstantDelaySchedule{Delay: intervalDuration}
		return schedule, nil
	}
	return nil, errors.New("calendar event must contain either a schedule or interval")
}
