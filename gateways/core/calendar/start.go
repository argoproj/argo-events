package calendar

import (
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	cronlib "github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

// StartConfig runs a configuration
func (ce *CalendarConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")
	cal, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
		return
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *cal).Msg("calendar configuration")

	go ce.fireEvent(cal, config)

	for {
		select {
		case <-config.StartChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
			config.Active = true

		case data := <-config.DataChan:
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

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

// fireEvent fires an event when schedule is passed.
func (ce *CalendarConfigExecutor) fireEvent(cal *CalSchedule, config *gateways.ConfigContext) {
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

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("k8 event created marking configuration as running")

	config.StartChan <- struct{}{}

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
		ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				config.ErrChan <- err
				return
			}
			config.DataChan <- payload
		case <-config.DoneChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
			config.ShutdownChan <- struct{}{}
			return
		}
	}
}
