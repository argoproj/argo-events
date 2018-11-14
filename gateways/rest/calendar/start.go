package calendar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	cronlib "github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"time"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

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

// StartGateway runs given configuration and sends event back to gateway processor client
func (ce *CalendarConfigExecutor) StartGateway(config *gateways.ConfigContext) {
	ce.GwConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")
	cal, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
		return
	}
	ce.GwConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *cal).Msg("calendar configuration")

	go ce.listenEvents(cal, config)

	for {
		select {
		case <-config.StartChan:
			b, err := json.Marshal(config.Data.Config)
			if err != nil {
				ce.GwConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to marshal config activated notification")
				_, err = http.Post(fmt.Sprintf("http://localhost:%s%s", ce.HttpClientPort, ce.ConfigErrorEndpoint), "application/octet-stream", bytes.NewReader([]byte(err.Error())))
				if err != nil {
					ce.GwConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to send config error notification")
				}
				return
			}
			_, err = http.Post(fmt.Sprintf("http://localhost:%s%s", ce.HttpClientPort, ce.ConfigActivatedEndpoint), "application/octet-stream", bytes.NewReader(b))
			if err != nil {
				ce.GwConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to send config activated notification")
			}

		case data := <-config.DataChan:
			ge := &gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			}

			payload, err := json.Marshal(ge)
			if err != nil {
				ce.GwConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to marshal event payload")
				_, err = http.Post(fmt.Sprintf("http://localhost:%s%s", ce.HttpClientPort, ce.ConfigErrorEndpoint), "application/octet-stream", bytes.NewReader([]byte(err.Error())))
				if err != nil {
					ce.GwConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to send config error notification")
				}
				return
			}

			ce.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway processor client")

			_, err = http.Post(fmt.Sprintf("http://localhost:%s%s", ce.HttpClientPort, ce.EventEndpoint), "application/octet-stream", bytes.NewReader(payload))
			if err != nil {
				config.ErrChan <- err
				return
			}

		case <-config.StopChan:
			ce.GwConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GwConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *CalendarConfigExecutor) listenEvents(cal *calSchedule, config *gateways.ConfigContext) {
	schedule, err := resolveSchedule(cal)
	if err != nil {
		config.ErrChan <- err
		return
	}

	exDates, err := common.ParseExclusionDates(cal.Recurrence)
	if err != nil {
		config.ErrChan <- err
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

	event := ce.GwConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	k8Event, err := common.CreateK8Event(event, ce.GwConfig.Clientset)
	if err != nil {
		ce.GwConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}
	ce.GwConfig.Log.Info().Str("config-key", config.Data.Src).Str("phase", string(v1alpha1.NodePhaseRunning)).Str("event-name", k8Event.Name).Msg("k8 event created")

	config.StartChan <- struct{}{}

	lastT := time.Now()

	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		ce.GwConfig.Log.Info().Str("config-key", config.Data.Src).Str("time", t.String()).Msg("expected next calendar event")

		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				if config.Active {
					config.Active = false
					config.ErrChan <- err
					return
				}
			}
			config.DataChan <- payload
		case <-config.DoneChan:
			return
		}
	}
}
