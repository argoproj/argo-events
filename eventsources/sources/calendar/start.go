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

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/persist"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for calendar based events
type EventListener struct {
	EventSourceName     string
	EventName           string
	Namespace           string
	CalendarEventSource v1alpha1.CalendarEventSource
	Metrics             *metrics.Metrics

	log              *zap.SugaredLogger
	eventPersistence persist.EventPersist
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
	el.log.Info("Initializing Persistence")
	if persistence.ConfigMap != nil {
		kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)

		restConfig, err := common.GetClientConfig(kubeConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to get a K8s rest config for the event source %s", el.GetEventName())
		}
		kubeClientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to set up a K8s client for the event source %s", el.GetEventName())
		}

		el.eventPersistence, err = persist.NewConfigMapPersist(ctx, kubeClientset, persistence.ConfigMap, el.Namespace)
		if err != nil {
			return err
		}
	}
	return nil
}

func (el *EventListener) getPersistenceKey() string {
	return fmt.Sprintf("%s.%s", el.EventSourceName, el.EventName)
}

// getExecutionTime return starting schedule time for execution
func (el *EventListener) getExecutionTime() (time.Time, error) {
	lastT := time.Now()
	if el.eventPersistence.IsEnabled() && el.CalendarEventSource.Persistence.IsCatchUpEnabled() {
		lastEvent, err := el.eventPersistence.Get(el.getPersistenceKey())
		if err != nil {
			el.log.Errorw("failed to get last persisted event.", zap.Error(err))
			return lastT, errors.Wrap(err, "failed to get last persisted event.")
		}
		if lastEvent != nil && lastEvent.EventPayload != "" {
			var eventData events.CalendarEventData
			err := json.Unmarshal([]byte(lastEvent.EventPayload), &eventData)
			if err != nil {
				el.log.Errorw("failed to marshal last persisted event.", zap.Error(err))
				return lastT, errors.Wrap(err, "failed to marshal last persisted event.")
			}
			eventTime := strings.Split(eventData.EventTime, " m=")
			lastT, err = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", eventTime[0])
			if err != nil {
				el.log.Errorw("failed to parse the persisted last event timestamp", zap.Error(err))
				return lastT, errors.Wrap(err, "failed to parse the persisted last event timestamp.")
			}
		}

		if el.CalendarEventSource.Persistence.Catchup.MaxDuration != "" {
			duration, err := time.ParseDuration(el.CalendarEventSource.Persistence.Catchup.MaxDuration)
			if err != nil {
				return lastT, err
			}

			// Set maxCatchupDuration in execution time if last persisted event time is greater than maxCatchupDuration
			if duration < time.Since(lastT) {
				el.log.Infow("set execution time", zap.Any("maxDuration", el.CalendarEventSource.Persistence.Catchup.MaxDuration))
				lastT = time.Now().Add(-duration)
			}
		}
	}
	return lastT, nil
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	el.log = logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
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

	el.eventPersistence = &persist.NullPersistence{}
	if calendarEventSource.Persistence != nil {
		if err = el.initializePersistence(ctx, calendarEventSource.Persistence); err != nil {
			return err
		}
	} else {
		el.log.Info("Persistence not enabled")
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

	lastT, err := el.getExecutionTime()
	if err != nil {
		return err
	}

	var location *time.Location
	if calendarEventSource.Timezone != "" {
		el.log.Infow("loading location for the schedule...", zap.Any("location", calendarEventSource.Timezone))
		location, err = time.LoadLocation(calendarEventSource.Timezone)
		if err != nil {
			return errors.Wrapf(err, "failed to load location for event source %s / %s", el.GetEventSourceName(), el.GetEventName())
		}
		lastT = lastT.In(location)
	}
	sendEventFunc := func(tx time.Time) error {
		startTime := time.Now()
		defer func(start time.Time) {
			elapsed := time.Now().Sub(start)
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(elapsed/time.Millisecond))
		}(startTime)

		eventData := &events.CalendarEventData{
			EventTime:   tx.String(),
			UserPayload: calendarEventSource.UserPayload,
			Metadata:    calendarEventSource.Metadata,
		}
		payload, err := json.Marshal(eventData)
		if err != nil {
			el.log.Errorw("failed to marshal the event data", zap.Error(err))
			// no need to continue as further event payloads will suffer same fate as this one.
			return errors.Wrapf(err, "failed to marshal the event data for event source %s / %s", el.GetEventSourceName(), el.GetEventName())
		}
		el.log.Info("dispatching calendar event...")
		err = dispatch(payload)
		if err != nil {
			el.log.Errorw("failed to dispatch calendar event", zap.Error(err))
			return errors.Wrapf(err, "failed to dispatch calendar event")
		}
		if el.eventPersistence != nil && el.eventPersistence.IsEnabled() {
			event := persist.Event{EventKey: el.getPersistenceKey(), EventPayload: string(payload)}
			err = el.eventPersistence.Save(&event)
			if err != nil {
				el.log.Errorw("failed to persist calendar event", zap.Error(err))
			}
		}
		return nil
	}

	el.log.Infow("Calendar event start time:", zap.Any("Time", lastT.Format(time.RFC822)))
	for {
		t := next(lastT)

		// Catchup scenario
		// Trigger the event immediately if the current schedule time is earlier then
		if time.Now().After(t) {
			el.log.Infow("triggering catchup events", zap.Any(logging.LabelTime, t.UTC().String()))
			if err = sendEventFunc(t); err != nil {
				el.log.Errorw("failed to dispatch calendar event", zap.Error(err))
				el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
				if el.eventPersistence.IsEnabled() {
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}
			lastT = t
			if location != nil {
				lastT = lastT.In(location)
			}
			continue
		}

		timer := time.After(time.Until(t))
		el.log.Infow("expected next calendar event", zap.Any(logging.LabelTime, t.UTC().String()))
		select {
		case tx := <-timer:
			if err = sendEventFunc(tx); err != nil {
				el.log.Errorw("failed to dispatch calendar event", zap.Error(err))
				el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
				if el.eventPersistence.IsEnabled() {
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}
			lastT = tx
			if location != nil {
				lastT = lastT.In(location)
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
