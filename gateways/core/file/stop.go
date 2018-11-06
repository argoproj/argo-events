package file

import "github.com/argoproj/argo-events/gateways"

// StopConfig deactivates a configuration
func (fw *FileWatcherConfigExecutor) StopConfig(config *gateways.ConfigContext) {
	if config.Active == true {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}
