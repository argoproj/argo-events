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

package minio

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/ghodss/yaml"
	"github.com/minio/minio-go"
)

// ValidateEventSource validates the minio event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	var sourceValue *apicommon.S3Artifact
	err := yaml.Unmarshal(eventSource.Value, &sourceValue)
	if err != nil {
		log.WithError(err).Error("failed to parse the minio event source")
		return &gateways.ValidEventSource{
			Reason:  err.Error(),
			IsValid: false,
		}, err
	}

	if err := validateMinioEventSource(sourceValue); err != nil {
		return &gateways.ValidEventSource{
			Reason:  err.Error(),
			IsValid: false,
		}, err
	}
	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

// validates the minio event source
func validateMinioEventSource(sourceValue *apicommon.S3Artifact) error {
	if sourceValue == nil {
		return gwcommon.ErrNilEventSource
	}
	if sourceValue.AccessKey == nil {
		return fmt.Errorf("access key can't be empty")
	}
	if sourceValue.SecretKey == nil {
		return fmt.Errorf("secret key can't be empty")
	}
	if sourceValue.Endpoint == "" {
		return fmt.Errorf("endpoint url can't be empty")
	}
	if sourceValue.Bucket != nil && sourceValue.Bucket.Name == "" {
		return fmt.Errorf("bucket name can't be empty")
	}
	if sourceValue.Events != nil {
		for _, event := range sourceValue.Events {
			if minio.NotificationEventType(event) == "" {
				return fmt.Errorf("unknown event %s", event)
			}
		}
	}
	return nil
}
