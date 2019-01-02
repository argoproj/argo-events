package kafka

import "github.com/argoproj/argo-events/gateways"

// StopConfiguration stops a configuration
func (kce *KafkaConfigExecutor) StopConfig(config *gateways.EventSourceContext) {
	if config.Active == true {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}
