package kafka

import "github.com/argoproj/argo-events/gateways"

// StopConfiguration stops a configuration
func (kce *KafkaConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}
