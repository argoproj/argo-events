package calendar

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/common"
	"time"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"fmt"
	cronlib "github.com/robfig/cron"
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Next is a function to compute the next signal time from a given time
type Next func(time.Time) time.Time

func parseConfig(config string) () {

}

// StartConfig runs a configuration
func (ce *CalendarConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var msg string

	// mark final gateway state
	defer ce.GatewayConfig.GatewayCleanup(config, &msg, err)

	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")
	var cal *CalSchedule
	err = mapstructure.Decode(config.Data.Config, &cal)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *cal).Msg("calendar configuration")

	go ce.fireEvent(cal, config)

	for {
		select {
		case <-ce.StartChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
			config.Active = true

		case data := <-ce.DataCh:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})

		case <-ce.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.Active = false
			ce.DoneCh <- struct{}{}
			return nil

		case err := <-ce.ErrChan:
			return err
		}
	}
	return nil
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
	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err := common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		ce.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("k8 event created marking configuration as running")
	ce.StartChan <- struct{}{}

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
		ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Str("time", t.String()).Msg("expected next calendar event")
		select {
		case tx := <-timer:
			lastT = tx
			event := metav1.Time{Time: t}
			payload, err := event.Marshal()
			if err != nil {
				ce.ErrChan <- err
				return
			}
			ce.DataCh <- payload
		case <-ce.DoneCh:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is stopped")
			return
		}
	}
}
