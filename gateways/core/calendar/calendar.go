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
	"fmt"
	"time"

	"bytes"
	"context"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
	hs "github.com/mitchellh/hashstructure"
	cronlib "github.com/robfig/cron"
	zlog "github.com/rs/zerolog"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
)

const (
	config = "calendar-gateway-configmap"
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

type calendar struct {
	gatewayConfig             *gateways.GatewayConfig
	registeredCalendarSignals map[uint64]*calSchedule
}

func (c *calendar) RunGateway(cm *apiv1.ConfigMap) error {
	for scheduleConfigKey, scheduleConfigValue := range cm.Data {
		var cal *calSchedule
		err := yaml.Unmarshal([]byte(scheduleConfigValue), &cal)
		if err != nil {
			c.gatewayConfig.Log.Error().Str("schedule-key", scheduleConfigKey).Err(err).Msg("failed to parse configuration")
			return err
		}
		key, err := hs.Hash(cal, &hs.HashOptions{})
		if err != nil {
			c.gatewayConfig.Log.Warn().Err(err).Msg("failed to get hash of configuration")
			continue
		}

		if _, ok := c.registeredCalendarSignals[key]; ok {
			c.gatewayConfig.Log.Info().Interface("config", *cal).Msg("duplicate configuration")
			continue
		}

		c.registeredCalendarSignals[key] = cal
		go c.startCalenderSchedule(cal)
	}
	return nil
}

func (c *calendar) startCalenderSchedule(cal *calSchedule) error {
	schedule, err := c.resolveSchedule(cal)
	if err != nil {
		return fmt.Errorf("failed to resolve calendar schedule. Err: %+v", err)
	}
	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		return fmt.Errorf("failed to resolve calendar schedule. Err: %+v", err)
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
		log.Printf("expected next calendar event %s", t)
		select {
		case tx := <-timer:
			lastT = tx
			c.gatewayConfig.Log.Info().Msg("event sending")
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				return fmt.Errorf("failed to marshal event. Err: %+v", err)
			} else {
				// dispatch the event to gateway transformer
				_, err = http.Post(fmt.Sprintf("http://localhost:%s", c.gatewayConfig.TransformerPort), "application/octet-stream", bytes.NewReader(payload))
				if err != nil {
					return fmt.Errorf("failed to dispatch the event. Err: %+v", err)
				}
				c.gatewayConfig.Log.Info().Msg("event dispatched to gateway transformer")
			}
		}
	}
}

func (c *calendar) getEventTimer(next Next) <-chan time.Time {
	lastT := time.Now()
	eventTimer := make(chan time.Time)
	go func() {
		defer close(eventTimer)
		for {
			t := next(lastT)
			timer := time.After(time.Until(t))
			log.Printf("expected next calendar event %s", t)
			select {
			case tx := <-timer:
				eventTimer <- tx
				lastT = tx
			}
		}
	}()
	return eventTimer
}

func (c *calendar) resolveSchedule(cal *calSchedule) (cronlib.Schedule, error) {
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
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	// Todo: hardcoded for now, move it to constants

	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}
	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)

	gatewayConfig := &gateways.GatewayConfig{
		Log:             zlog.New(os.Stdout).With().Logger(),
		Namespace:       namespace,
		Clientset:       clientset,
		TransformerPort: transformerPort,
	}
	cal := &calendar{
		registeredCalendarSignals: make(map[uint64]*calSchedule),
		gatewayConfig:             gatewayConfig,
	}

	_, err = gatewayConfig.WatchGatewayConfigMap(cal, context.Background(), config)
	if err != nil {
		panic(fmt.Errorf("failed to watch calendar schedules. Err: %+v", err))
	}
	// wait forever
	select {}
}
