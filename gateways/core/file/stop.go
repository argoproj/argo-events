package file

import "github.com/argoproj/argo-events/gateways"

// StopConfig deactivates a configuration
func (fw *FileWatcherConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}
