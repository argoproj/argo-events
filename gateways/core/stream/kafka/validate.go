package kafka

import (
	"github.com/argoproj/argo-events/gateways"
	"fmt"
)

// Validate validates the gateway configuration
func (kce *KafkaConfigExecutor) Validate(config *gateways.ConfigContext) error {
	kafkaConfig, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if kafkaConfig == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}
	if kafkaConfig.URL == "" {
		return fmt.Errorf("%+v, url must be specified", gateways.ErrInvalidConfig)
	}
	if kafkaConfig.Topic == "" {
		return fmt.Errorf("%+v, topic must be specified", gateways.ErrInvalidConfig)
	}
	if kafkaConfig.Partition == "" {
		return fmt.Errorf("%+v, partition must be specified", gateways.ErrInvalidConfig)
	}
	return nil
}
