package calendar

import "github.com/argoproj/argo-events/gateways"

// StopConfig deactivates a configuration
func (ce *CalendarConfigExecutor) StopConfig(config *gateways.ConfigContext) {
	if config.Active == true {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}
