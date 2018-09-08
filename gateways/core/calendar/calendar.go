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

package main

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core"
	"github.com/ghodss/yaml"
	cronlib "github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

// calSchedule describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type calSchedule struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule" protobuf:"bytes,1,opt,name=schedule"`

	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval" protobuf:"bytes,2,opt,name=interval"`

	// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
	// RDATE defines the list of DATE-TIME values for recurring events.
	// EXDATE defines the list of DATE-TIME exceptions for recurring events.
	// the combination of these rules and dates combine to form a set of date times.
	// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
	Recurrence []string `json:"recurrence,omitempty" protobuf:"bytes,3,rep,name=recurrence"`
}

// Runs a configuration
func configRunner(config *gateways.ConfigData) error {
	gatewayConfig.Log.Info().Str("config-name", config.Src).Msg("parsing configuration...")

	var cal *calSchedule
	err := yaml.Unmarshal([]byte(config.Config), &cal)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to parse configuration")
		return err
	}

	schedule, err := resolveSchedule(cal)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to resolve calendar schedule.")
		return err
	}
	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to resolve calendar schedule.")
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

	gatewayConfig.Log.Info().Str("config-name", config.Src).Msg("configuration is running...")
	config.Active = true
	lastT := time.Now()
calendarLoop:
	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		gatewayConfig.Log.Info().Str("config-name", config.Src).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				gatewayConfig.Log.Error().Err(err).Msg("failed to marshal event")
			} else {
				gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("dispatching event to gateway-processor")
				gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src: config.Src,
					Payload: payload,
				})
			}
		case <-config.StopCh:
			gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("stopping the configuration...")
			break calendarLoop
		}
	}
	return nil
}

func resolveSchedule(cal *calSchedule) (cronlib.Schedule, error) {
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

func main() {
	gatewayConfig.WatchGatewayConfigMap(context.Background(), configRunner, core.ConfigDeactivator)
	select {}
}
