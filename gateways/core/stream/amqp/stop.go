package amqp

import "github.com/argoproj/argo-events/gateways"

// StopConfig stops a configuration
func (ace *AMQPConfigExecutor) StopConfig(config *gateways.EventSourceContext) {
	if config.Active == true {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}
