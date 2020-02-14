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
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	cronlib "github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

// EventListener implements Eventing for calendar based events
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
}

// Next is a function to compute the next event time from a given time
type Next func(time.Time) time.Time

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	channels := server.NewChannels()

	go server.HandleEventsFromEventSource(eventSource.Name, eventStream, channels, listener.Logger)

	defer func() {
		channels.Stop <- struct{}{}
	}()

	if err := listener.listenEvents(eventSource, channels); err != nil {
		listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to listen to events")
		return err
	}

	return nil
}

// listenEvents fires an event when schedule completes.
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing calendar event source...")
	var calendarEventSource *v1alpha1.CalendarEventSource
	if err := yaml.Unmarshal(eventSource.Value, &calendarEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse event source %s", eventSource.Name)
	}

	logger.Infoln("resolving calendar schedule...")
	schedule, err := resolveSchedule(calendarEventSource)
	if err != nil {
		return err
	}

	logger.Infoln("parsing exclusion dates if any...")
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
		logger.WithField("location", calendarEventSource.Timezone).Infoln("loading location for the schedule...")
		location, err = time.LoadLocation(calendarEventSource.Timezone)
		if err != nil {
			return errors.Wrapf(err, "failed to load location for event source %s", eventSource.Name)
		}
		lastT = lastT.In(location)
	}

	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		logger.WithField(common.LabelTime, t.UTC().String()).Info("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			if location != nil {
				lastT = lastT.In(location)
			}
			response := &apicommon.CalendarEventData{
				EventTime:   tx.String(),
				UserPayload: calendarEventSource.UserPayload,
			}
			payload, err := json.Marshal(response)
			if err != nil {
				// no need to continue as further event payloads will suffer same fate as this one.
				return errors.Wrapf(err, "failed to marshal the event data for event source %s", eventSource.Name)
			}
			logger.Infoln("event dispatched on data channel")
			channels.Data <- payload
		case <-channels.Done:
			return nil
		}
	}
}

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
	} else if cal.Interval != "" {
		intervalDuration, err := time.ParseDuration(cal.Interval)
		if err != nil {
			return nil, errors.Errorf("failed to parse interval %s from calendar event. Cause: %+v", cal.Interval, err.Error())
		}
		schedule := cronlib.ConstantDelaySchedule{Delay: intervalDuration}
		return schedule, nil
	} else {
		return nil, errors.New("calendar event must contain either a schedule or interval")
	}
}
