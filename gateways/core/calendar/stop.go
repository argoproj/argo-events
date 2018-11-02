package calendar

import "github.com/argoproj/argo-events/gateways"

// StopConfig deactivates a configuration
func (ce *CalendarConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}
