package artifact

import "github.com/argoproj/argo-events/gateways"

// StopConfig stops the configuration
func (s3ce *S3ConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}
