package amazon_sns

import (
	"github.com/argoproj/argo-events/gateways"
	"fmt"
)

func (ce *AWSSNSConfigExecutor) Validate(config *gateways.ConfigContext) error {
	snsConfig, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if snsConfig.SecretKey == nil {
		return fmt.Errorf("secret key is not provided")
	}
	if snsConfig.AccessKey == nil {
		return fmt.Errorf("access key is not provided")
	}
	if snsConfig.Region == "" {
		return fmt.Errorf("region is not provided")
	}
	if snsConfig.AccessToken == nil {
		return fmt.Errorf("access token is not provided")
	}
	return nil
}
