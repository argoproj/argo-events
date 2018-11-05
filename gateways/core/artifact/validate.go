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
	"github.com/ghodss/yaml"
)

// Validate validates s3 configuration
func(s3ce *S3ConfigExecutor) Validate(config *gateways.ConfigContext) error {
	artifact, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if artifact == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3EventConfig == nil {
		return fmt.Errorf("%+v, s3 bucket configuration can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.AccessKey == nil {
		return fmt.Errorf("%+v, access key can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.SecretKey == nil {
		return fmt.Errorf("%+v, secret key can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3EventConfig.Endpoint == "" {
		return fmt.Errorf("%+v, endpoint url can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3EventConfig.Bucket == "" {
		return fmt.Errorf("%+v, bucket name can't be empty", gateways.ErrInvalidConfig)
	}
	if artifact.S3EventConfig.Event != "" && minio.NotificationEventType(artifact.S3EventConfig.Event) == "" {
		return fmt.Errorf("%+v, unknown event %s", gateways.ErrInvalidConfig, artifact.S3EventConfig.Event)
	}
	return nil
}

func parseConfig(config string) (*S3Artifact, error) {
	var i *S3Artifact
	err := yaml.Unmarshal([]byte(config), &i)
	if err != nil {
		return nil, err
	}
	return i, err
}
