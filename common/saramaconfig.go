package common

import (
	"bytes"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/spf13/viper"
)

// GetSaramaConfigFromYAMLString parse yaml string to sarama.config.
// Note: All the time.Duration config can not be correctly decoded because it does not implement the decode function.
func GetSaramaConfigFromYAMLString(yaml string) (*sarama.Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(yaml)); err != nil {
		return nil, err
	}
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed validating sarama config, %w", err)
	}
	return cfg, nil
}
