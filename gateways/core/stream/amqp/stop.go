package amqp

import "github.com/argoproj/argo-events/gateways"

// StopConfig stops a configuration
func (ace *AMQPConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopChan <- struct{}{}
	}
	return nil
}
