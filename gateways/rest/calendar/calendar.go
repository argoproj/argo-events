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

	"github.com/argoproj/argo-events/common"
	"github.com/ghodss/yaml"
	cronlib "github.com/robfig/cron"
	zlog "github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/argoproj/argo-events/gateways"
	"net/http"
	"bytes"
	"os"
	"sync"
	"encoding/json"
	"io/ioutil"
	"github.com/argoproj/argo-events/gateways/utils"
	"io"
)

var (
	log = zlog.New(os.Stdout).With().Str("gateway", "calendar").Logger()

	httpServerPort = func() string {
		httpServerPort, ok := os.LookupEnv(common.GatewayProcessorServerHTTPPortEnvVar)
		if !ok {
			panic("gateway server http port is not provided")
		}
		return httpServerPort
	}()

	httpClientPort = func() string {
		httpClientPort, ok := os.LookupEnv(common.GatewayProcessorClientHTTPPortEnvVar)
		if !ok {
			panic("gateway client http port is not provided")
		}
		return httpClientPort
	}()

	configStartEndpoint = func() string {
		configStartEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerConfigStartEndpointEnvVar)
		if !ok {
			panic("gateway config start endpoint is not provided")
		}
		return configStartEndpoint
	}()

	configStopEndpoint = func() string {
		configStopEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerConfigStopEndpointEnvVar)
		if !ok {
			panic("gateway config stop endpoint is not provided")
		}
		return configStopEndpoint
	}()

	eventEndpoint = func() string {
		eventEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerEventEndpointEnvVar)
		if !ok {
			panic("gateway event post endpoint is not provided")
		}
		return eventEndpoint
	}()

	activeConfigs = make(map[uint64]*gateways.ConfigData)

	mut sync.Mutex
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

// calSchedule describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type calSchedule struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string

	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string

	// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
	// RDATE defines the list of DATE-TIME values for recurring events.
	// EXDATE defines the list of DATE-TIME exceptions for recurring events.
	// the combination of these rules and dates combine to form a set of date times.
	// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
	Recurrence []string
}

func runGateway(config *gateways.ConfigData) error {
	log.Info().Str("config-name", config.Src).Msg("parsing calendar schedule...")
	var cal *calSchedule
	err := yaml.Unmarshal([]byte(config.Config), &cal)
	if err != nil {
		log.Error().Str("config-key", config.Src).Err(err).Msg("failed to parse calendar schedule")
		return err
	}

	schedule, err := resolveSchedule(cal)
	if err != nil {
		log.Error().Str("config-key", config.Src).Err(err).Msg("failed to resolve calendar schedule.")
		return err
	}
	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		log.Error().Str("config-key", config.Src).Err(err).Msg("failed to resolve calendar schedule.")
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
calendarConfigRunner:
	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		log.Info().Str("config-key", config.Src).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal event")
			} else {
				re := gateways.GatewayEvent{
					Src: config.Src,
					Payload: payload,
				}
				payload, err = json.Marshal(re)
				if err != nil {
					log.Error().Err(err).Msg("failed to marshal payload into gateway event")
					continue
				}
				_, err := http.Post(fmt.Sprintf("http://localhost:%s%s", httpClientPort, eventEndpoint), "application/octet-stream", bytes.NewReader(payload))
				if err != nil {
					log.Warn().Str("config-key", config.Src).Err(err).Msg("failed to dispatch event to gateway-processor.")
				}
			}
		case <-config.StopCh:
			log.Info().Str("config-key", config.Src).Msg("stopping configuration")
			break calendarConfigRunner
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

func getConfiguration(body io.ReadCloser) (*gateways.ConfigData, *uint64, error) {
	var configData gateways.HTTPGatewayConfig
	config, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read requested config to run. err %+v", err)
	}
	err = json.Unmarshal(config, &configData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config. err %+v", err)
	}
	// register configuration
	gatewayConfig := &gateways.ConfigData{
		Config: configData.Config,
		Src: configData.Src,
		StopCh: make(chan struct{}),
	}
	hash, err := utils.Hasher(gatewayConfig.Src, gatewayConfig.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash configuration. err %+v", err)
	}
	return gatewayConfig, &hash, nil
}

func main() {
	http.HandleFunc(configStartEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		config, hash, err := getConfiguration(request.Body)
		if err != nil {
			log.Error().Err(err).Msg("failed to start configuration")
			common.SendErrorResponse(writer)
			return
		}
		mut.Lock()
		activeConfigs[*hash] = config
		mut.Unlock()
		go runGateway(config)
	})
	http.HandleFunc(configStopEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		_, hash, err := getConfiguration(request.Body)
		if err != nil {
			log.Error().Err(err).Msg("failed to stop configuration")
			common.SendErrorResponse(writer)
			return
		}
		mut.Lock()
		config := activeConfigs[*hash]
		config.StopCh <- struct{}{}
		delete(activeConfigs, *hash)
		mut.Unlock()
	})
	log.Fatal().Str("port", httpServerPort).Err(http.ListenAndServe(":"+fmt.Sprintf("%s",  httpServerPort), nil)).Msg("gateway server started listening")
}
