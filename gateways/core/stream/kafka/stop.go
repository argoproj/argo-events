package kafka

import "github.com/argoproj/argo-events/gateways"

// StopConfiguration stops a configuration
func (kce *KafkaConfigExecutor) StopConfig(config *gateways.ConfigContext) {
	if config.Active == true {
		config.Active = true
		config.StopChan <- struct{}{}
	}
}
