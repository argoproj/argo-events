package nats

import "github.com/argoproj/argo-events/gateways"

// StopConfig stops gateway configuration
func (nce *NatsConfigExecutor) StopConfig(config *gateways.EventSourceContext) {
	if config.Active == true {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}
