package calendar

import (
	"github.com/argoproj/argo-events/gateways"
	"fmt"
	"github.com/mitchellh/mapstructure"
)

// Validate validates gateway configuration
func (ce *CalendarConfigExecutor) Validate(config *gateways.ConfigContext) error {
	var cal *CalSchedule
	err := mapstructure.Decode(config.Data.Config, &cal)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if cal == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}
	if cal.Schedule == "" && cal.Interval == "" {
		return fmt.Errorf("%+v, must have either schedule or interval", gateways.ErrInvalidConfig)
	}
	_, err = resolveSchedule(cal)
	if err != nil {
		return err
	}
	return nil
}
