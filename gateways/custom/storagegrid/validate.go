package storagegrid

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	"strings"
)

// Validate validates gateway configuration
func (sgce *StorageGridConfigExecutor) Validate(config *gateways.ConfigContext) error {
	sg, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if sg == nil {
		return gateways.ErrEmptyConfig
	}
	if sg.Port == "" {
		return fmt.Errorf("%+v, must specify port", gateways.ErrInvalidConfig)
	}
	if sg.Endpoint == "" {
		return fmt.Errorf("%+v, must specify endpoint", gateways.ErrInvalidConfig)
	}
	if !strings.HasPrefix(sg.Endpoint, "/") {
		return fmt.Errorf("%+v, endpoint must start with '/'", gateways.ErrInvalidConfig)
	}
	return nil
}
