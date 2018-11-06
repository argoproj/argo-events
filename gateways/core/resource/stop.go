package resource

import "github.com/argoproj/argo-events/gateways"

// StopConfig stops a configuration
func (rce *ResourceConfigExecutor) StopConfig(config *gateways.ConfigContext) {
	if config.Active == true {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}
