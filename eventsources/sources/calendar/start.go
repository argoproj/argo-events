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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	cronlib "github.com/robfig/cron"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/persist"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for calendar based events
type EventListener struct {
	EventSourceName     string
	EventName           string
	CalendarEventSource v1alpha1.CalendarEventSource
	log                 *zap.Logger
	EventPersistence    persist.EventPersist
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

// initializePersistence initialize the persistence object.
// This func can move to eventing.go once we start supporting persistence for all sources.
func (el *EventListener) initializePersistence(ctx context.Context, persistence *v1alpha1.EventPersistence) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName()).Desugar()
	log.Info("Initializing Persistence")
	if persistence.ConfigMap != nil {
		config := config.GetConfigOrDie()
		kubeclientset := kubernetes.NewForConfigOrDie(config)
		namespace, defined := os.LookupEnv("POD_NAMESPACE")
		if !defined {
			log.Fatal("required environment variable 'POD_NAMESPACE' not defined")
		}
		log.Info(namespace)
		var err error
		el.EventPersistence, err = persist.NewConfigMapPersist(kubeclientset, persistence.ConfigMap, namespace)
		if err != nil {
			return err
		}
	}
	return nil
}
func (el *EventListener) GetPersistenceKey() string{
	return fmt.Sprintf("%s.%s", el.EventSourceName, el.EventName)
}

func (el *EventListener) GetStartingTime() time.Time {
	lastT := time.Now()
	if el.CalendarEventSource.Catchup && el.EventPersistence.IsEnabled() {
		lastEvent, err := el.EventPersistence.Get(el.GetPersistenceKey())
		if err != nil {
			el.log.Error("failed to get last persisted events. ", zap.Error(err))
		}
		if lastEvent != nil && lastEvent.EventPayload != "" {
			var eventData events.CalendarEventData
			err := json.Unmarshal([]byte(lastEvent.EventPayload), &eventData)
			if err != nil {
				el.log.Info(" ", zap.Error(err))
			}
			eventTime := strings.Split(eventData.EventTime, " m=")
			lastT, err = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", eventTime[0])
			if err != nil {
				el.log.Error("failed to parse the persisted last event timestamp", zap.Error(err))
				lastT = time.Now()
			}
		}
	}
	return lastT
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	el.log = logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName()).Desugar()
	el.log.Info("started processing the calendar event source...")

	calendarEventSource := &el.CalendarEventSource
	el.log.Info("resolving calendar schedule...")
	schedule, err := resolveSchedule(calendarEventSource)
	if err != nil {
		return err
	}

	el.log.Info("parsing exclusion dates if any...")
	exDates, err := common.ParseExclusionDates(calendarEventSource.ExclusionDates)
	if err != nil {
		return err
	}

	el.EventPersistence = &persist.NullPersistence{}
	if calendarEventSource.Persistence != nil {
		err = el.initializePersistence(ctx, calendarEventSource.Persistence)
		if err != nil {
			return err
		}
	} else {
		el.log.Info("Persistence disabled")
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

	lastT := el.GetStartingTime()

	var location *time.Location
	if calendarEventSource.Timezone != "" {
		el.log.Info("loading location for the schedule...", zap.Any("location", calendarEventSource.Timezone))
		location, err = time.LoadLocation(calendarEventSource.Timezone)
		if err != nil {
			return errors.Wrapf(err, "failed to load location for event source %s / %s", el.GetEventSourceName(), el.GetEventName())
		}
		lastT = lastT.In(location)
	}

	sendEventFunc := func(tx time.Time) error {
		response := &events.CalendarEventData{
			EventTime:   tx.String(),
			UserPayload: calendarEventSource.UserPayload,
			Metadata:    calendarEventSource.Metadata,
		}
		payload, err := json.Marshal(response)
		if err != nil {
			el.log.Error("failed to marshal the event data", zap.Error(err))
			// no need to continue as further event payloads will suffer same fate as this one.
			return errors.Wrapf(err, "failed to marshal the event data for event source %s / %s", el.GetEventSourceName(), el.GetEventName())
		}
		el.log.Info("dispatching calendar event...")
		err = dispatch(payload)
		if err != nil {
			el.log.Error("failed to dispatch calendar event", zap.Error(err))
		}
		event := persist.Event{EventKey: el.GetPersistenceKey(), EventPayload: string(payload)}
		err = el.EventPersistence.Save(&event)
		if err != nil {
			el.log.Error("failed to dispatch calendar event", zap.Error(err))
		}
		return nil
	}

	el.log.Info("Calendar event start time:", zap.Any("Time", lastT.Format(time.RFC822)))
	for {
		t := next(lastT)

		// Catchup scenario
		// Trigger the event immediately if the current schedule time is earlier then
		if time.Now().After(t) {
			el.log.Info("triggering catchup events", zap.Any(logging.LabelTime, t.UTC().String()))
			lastT = t
			if location != nil {
				lastT = lastT.In(location)
			}
			err = sendEventFunc(t)
			if err != nil {
			}
			continue
		}

		timer := time.After(time.Until(t))
		el.log.Info("expected next calendar event", zap.Any(logging.LabelTime, t.UTC().String()))
		select {
		case tx := <-timer:
			lastT = tx
			if location != nil {
				lastT = lastT.In(location)
			}
			err = sendEventFunc(tx)
			if err != nil {

			}
		case <-ctx.Done():
			el.log.Info("exiting calendar event listener...")
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
