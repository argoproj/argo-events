/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package artifact

import (
	"github.com/argoproj/argo-events/gateways"
	"fmt"
	"github.com/minio/minio-go"
	"github.com/mitchellh/mapstructure"
	sv1alphav1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// Validate validates s3 configuration
func(s3ce *S3ConfigExecutor) Validate(config *gateways.ConfigContext) error {
	var artifact *sv1alphav1.S3Artifact
	err := mapstructure.Decode(config.Data.Config, &artifact)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if artifact == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3Bucket == nil {
		return fmt.Errorf("%+v, s3 bucket configuration can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3Bucket.AccessKey == nil {
		return fmt.Errorf("%+v, access key can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3Bucket.SecretKey == nil {
		return fmt.Errorf("%+v, secret key can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3Bucket.Endpoint == "" {
		return fmt.Errorf("%+v, endpoint url can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3Bucket.Bucket == "" {
		return fmt.Errorf("%+v, bucket name can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.Event != "" && minio.NotificationEventType(artifact.Event) == "" {
		return fmt.Errorf("%+v, unknown event %s", gateways.ErrInvalidConfig, artifact.Event)
	}
	return nil
}
