package amqp

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

// Validate validates gateway configuration
func (ace *AMQPConfigExecutor) Validate(config *gateways.ConfigContext) error {
	a, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if a == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}
	if a.URL == "" {
		return fmt.Errorf("%+v, url must be specified", gateways.ErrInvalidConfig)
	}
	if a.RoutingKey == "" {
		return fmt.Errorf("%+v, routing key must be specified", gateways.ErrInvalidConfig)
	}
	if a.ExchangeName == "" {
		return fmt.Errorf("%+v, exchange name must be specified", gateways.ErrInvalidConfig)
	}
	if a.ExchangeType == "" {
		return fmt.Errorf("%+v, exchange type must be specified", gateways.ErrInvalidConfig)
	}
	return nil
}
