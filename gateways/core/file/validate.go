package file

import (
	"github.com/argoproj/argo-events/gateways"
	"fmt"
	"github.com/mitchellh/mapstructure"
)

// Validate validates gateway configuration
func (fw *FileWatcherConfigExecutor) Validate(config *gateways.ConfigContext) error {
	var fwc *FileWatcherConfig
	err := mapstructure.Decode(config.Data.Config, &fwc)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if fwc == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}
	if fwc.Type == "" {
		return fmt.Errorf("%+v, type must be specified", gateways.ErrInvalidConfig)
	}
	if fwc.Directory == "" {
		return fmt.Errorf("%+v, directory must be specified", gateways.ErrInvalidConfig)
	}
	if fwc.Path == "" {
		return fmt.Errorf("%+v, path must be specified", gateways.ErrInvalidConfig)
	}
	return nil
}
