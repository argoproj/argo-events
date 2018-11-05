package nats

import "github.com/argoproj/argo-events/gateways"

// StopConfig stops gateway configuration
func (nce *NatsConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopChan <- struct{}{}
	}
	return nil
}
