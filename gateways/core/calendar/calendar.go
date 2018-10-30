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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
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

// calendarConfigExecutor implements ConfigExecutor interface
type calendarConfigExecutor struct{}

// calSchedule describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type calSchedule struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule"`

	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval"`

	// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
	// RDATE defines the list of DATE-TIME values for recurring events.
	// EXDATE defines the list of DATE-TIME exceptions for recurring events.
	// the combination of these rules and dates combine to form a set of date times.
	// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
	Recurrence []string `json:"recurrence,omitempty"`
}

// StartConfig runs a configuration
func (ce *calendarConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")

	var cal *calSchedule
	err = yaml.Unmarshal([]byte(config.Data.Config), &cal)
	if err != nil {
		errMessage = "failed to parse configuration"
		return err
	}

	schedule, err := resolveSchedule(cal)
	if err != nil {
		errMessage = "failed to resolve calendar schedule."
		return err
	}
	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		errMessage = "failed to resolve calendar schedule."
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

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running...")
	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("k8 event created marking configuration as running")

	lastT := time.Now()
calendarLoop:
	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				errMessage = "failed to marshal event"
				config.StopCh <- struct{}{}
				break calendarLoop
			} else {
				gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
				gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: payload,
				})
			}
		case <-config.StopCh:
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")
			config.Active = false
			break calendarLoop
		}
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// StopConfig deactivates a configuration
func (ce *calendarConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

func resolveSchedule(cal *calSchedule) (cronlib.Schedule, error) {
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

func main() {
	err := gatewayConfig.TransformerReadinessProbe()
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayTransformerConnection)
	}
	_, err = gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayEventWatch)
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &calendarConfigExecutor{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayConfigmapWatch)
	}
}
