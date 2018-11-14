package storagegrid

import "github.com/argoproj/argo-events/gateways"

// StopConfig stops the configuration
func (sgce *StorageGridConfigExecutor) StopConfig(config *gateways.ConfigContext) {
	if config.Active == true {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}
