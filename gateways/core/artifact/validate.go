package artifact

import (
	"github.com/argoproj/argo-events/gateways"
	"fmt"
	"github.com/minio/minio-go"
	"github.com/mitchellh/mapstructure"
)

// Validate validates s3 configuration
func(s3ce *S3ConfigExecutor) Validate(config *gateways.ConfigContext) error {
	var artifact *S3Artifact
	err := mapstructure.Decode(config.Data.Config, &artifact)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if artifact.AccessKey == nil {
		return fmt.Errorf("access key can't be empty")
	}
	if artifact.SecretKey == nil {
		return fmt.Errorf("secret key can't be empty")
	}
	if artifact.S3EventConfig == nil {
		return fmt.Errorf("s3 event configuration can't be empty")
	}
	if artifact.S3EventConfig.Endpoint == "" {
		return fmt.Errorf("endpoint url can't be empty")
	}
	if artifact.S3EventConfig.Bucket == "" {
		return fmt.Errorf("bucket name can't be empty")
	}
	if artifact.S3EventConfig.Event != "" && minio.NotificationEventType(artifact.S3EventConfig.Event) == "" {
		return fmt.Errorf("unknown event %s", artifact.S3EventConfig.Event)
	}
	return nil
}
