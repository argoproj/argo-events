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
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	cronlib "github.com/robfig/cron"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sync"
)

var (
	// gateway http server configurations
	httpGatewayServerConfig = gateways.NewHTTPGatewayServerConfig()
)

var (
	mut sync.Mutex
	// activeConfigs keeps track of configurations that are running in gateway.
	activeConfigs = make(map[string]*gateways.ConfigContext)
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

// runs given configuration and sends event back to gateway processor client
func runGateway(config *gateways.ConfigContext) error {
	var err error
	var errMsg string

	defer func() {
		config.Active = false
		if err != nil {
			httpGatewayServerConfig.GwConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg(errMsg)
			httpGatewayServerConfig.GwConfig.MarkGatewayNodePhase(config.Data.Src, v1alpha1.NodePhaseError, err.Error())
		} else {
			httpGatewayServerConfig.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration completed")
			httpGatewayServerConfig.GwConfig.MarkGatewayNodePhase(config.Data.Src, v1alpha1.NodePhaseCompleted, "configuration completed")
		}
		httpGatewayServerConfig.GwConfig.PersistUpdates()
	}()

	httpGatewayServerConfig.GwConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing calendar schedule...")

	var cal *calSchedule
	err = yaml.Unmarshal([]byte(config.Data.Config), &cal)
	if err != nil {
		errMsg = "failed to parse calendar schedule"
		return err
	}

	schedule, err := resolveSchedule(cal)
	if err != nil {
		errMsg = "failed to resolve calendar schedule."
		return err
	}
	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		errMsg = "failed to resolve calendar schedule."
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
		httpGatewayServerConfig.GwConfig.Log.Info().Str("config-key", config.Data.Src).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				httpGatewayServerConfig.GwConfig.Log.Error().Err(err).Msg("failed to marshal event")
			} else {
				re := gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: payload,
				}
				payload, err = json.Marshal(re)

				if err != nil {
					errMsg = "failed to marshal payload into gateway event"
					return err
				}
				httpGatewayServerConfig.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway processor client")
				httpGatewayServerConfig.GwConfig.Log.Info().Str("url", fmt.Sprintf("http://localhost:%s%s", httpGatewayServerConfig.HTTPClientPort, httpGatewayServerConfig.EventEndpoint)).Msg("client url")
				_, err := http.Post(fmt.Sprintf("http://localhost:%s%s", httpGatewayServerConfig.HTTPClientPort, httpGatewayServerConfig.EventEndpoint), "application/octet-stream", bytes.NewReader(payload))
				if err != nil {
					errMsg = "failed to dispatch event to gateway-processor."
					return err
				}
			}
		case <-config.StopCh:
			httpGatewayServerConfig.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping configuration")
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

// returns a gateway configuration and its hash
func getConfiguration(body io.ReadCloser) (*gateways.ConfigContext, *string, error) {
	var configData gateways.ConfigData
	config, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read requested config to run. err %+v", err)
	}
	err = json.Unmarshal(config, &configData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config. err %+v", err)
	}
	// register configuration
	gatewayConfig := &gateways.ConfigContext{
		Data: &gateways.ConfigData{
			Config: configData.Config,
			Src:    configData.Src,
		},
		StopCh: make(chan struct{}),
	}
	hash := gateways.Hasher(configData.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash configuration. err %+v", err)
	}
	return gatewayConfig, &hash, nil
}

func main() {
	// handles new configuration. adds a stop channel to configuration, so we can pass stop signal
	// in the event of configuration deactivation
	http.HandleFunc(httpGatewayServerConfig.ConfigActivateEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		config, hash, err := getConfiguration(request.Body)
		if err != nil {
			httpGatewayServerConfig.GwConfig.Log.Error().Err(err).Msg("failed to start configuration")
			common.SendErrorResponse(writer)
			return
		}
		mut.Lock()
		activeConfigs[*hash] = config
		mut.Unlock()
		common.SendSuccessResponse(writer)
		go runGateway(config)
	})

	// handles configuration deactivation. no need to remove the configuration from activeConfigs as we are
	// always overriding configurations in configuration activation.
	http.HandleFunc(httpGatewayServerConfig.ConfigurationDeactivateEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		_, hash, err := getConfiguration(request.Body)
		if err != nil {
			httpGatewayServerConfig.GwConfig.Log.Error().Err(err).Msg("failed to stop configuration")
			common.SendErrorResponse(writer)
			return
		}
		mut.Lock()
		config := activeConfigs[*hash]
		config.StopCh <- struct{}{}
		delete(activeConfigs, *hash)
		mut.Unlock()
		common.SendSuccessResponse(writer)
	})

	// start http server
	httpGatewayServerConfig.GwConfig.Log.Fatal().Str("port", httpGatewayServerConfig.HTTPServerPort).Err(http.ListenAndServe(":"+fmt.Sprintf("%s", httpGatewayServerConfig.HTTPServerPort), nil)).Msg("gateway server started listening")
}
