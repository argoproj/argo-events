package resource

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

// Validate validates gateway configuration
func (rce *ResourceConfigExecutor) Validate(config *gateways.ConfigContext) error {
	res, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if res == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}

	if res.Version == "" {
		return fmt.Errorf("%+v, resource version must be specified", gateways.ErrInvalidConfig)
	}
	if res.Namespace == "" {
		return fmt.Errorf("%+v, resource namespace must be specified", gateways.ErrInvalidConfig)
	}
	if res.Kind == "" {
		return fmt.Errorf("%+v, resource kind must be specified", gateways.ErrInvalidConfig)
	}
	if res.Group == "" {
		return fmt.Errorf("%+v, resource group must be specified", gateways.ErrInvalidConfig)
	}
	return nil
}
