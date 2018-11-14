package nats

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

// Validate validates gateway configuration
func (nce *NatsConfigExecutor) Validate(config *gateways.ConfigContext) error {
	n, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if n == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}
	if n.URL == "" {
		return fmt.Errorf("%+v, url must be specified", gateways.ErrInvalidConfig)
	}
	if n.Subject == "" {
		return fmt.Errorf("%+v, subject must be specified", gateways.ErrInvalidConfig)
	}
	return nil
}
