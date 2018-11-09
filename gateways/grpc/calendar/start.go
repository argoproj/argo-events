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
	cronlib "github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gp "github.com/argoproj/argo-events/gateways/grpc/proto"
	"time"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

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

// StartConfig runs a configuration
func (ce *CalendarConfigExecutor) StartConfig(config *gp.GatewayConfig, eventStream gp.GatewayExecutor_StartConfigServer) error {
	ce.GatewayConfig.Log.Info().Str("config-name", config.Src).Msg("parsing configuration...")
	cal, err := parseConfig(config.Config)
	if err != nil {
		return err
	}

	ce.GatewayConfig.Log.Debug().Str("config-key", config.Src).Interface("config-value", *cal).Msg("calendar configuration")

	err = ce.Validate(config)
	if err != nil {
		return err
	}

	go ce.fireEvent(cal, config)

	for {
		select {
		case data := <-ce.DataChan:
			eventStream.Send(&gp.Event{
				Data: data,
			})

		case err := <-ce.ErrChan:
			return err

		case <-eventStream.Context().Done():
			ce.StopChan <- struct{}{}
			return nil
		}
	}
}

// fireEvent fires an event when schedule is passed.
func (ce *CalendarConfigExecutor) fireEvent(cal *CalSchedule, config *gp.GatewayConfig) {
	schedule, err := resolveSchedule(cal)
	if err != nil {
		ce.ErrChan <- err
		return
	}

	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		ce.ErrChan <- err
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
		ce.GatewayConfig.Log.Info().Str("config-name", config.Src).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				ce.ErrChan <- err
				return
			}
			ce.DataChan <- payload
		case <-ce.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Src).Msg("configuration stopped")
			return
		}
	}
}
