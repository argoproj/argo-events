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

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/ghodss/yaml"
	"github.com/minio/minio-go"
)

// ValidateEventSource validates the minio event source
func (listener *EventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	if apicommon.EventSourceType(eventSource.Type) != apicommon.MinioEvent {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  common.ErrEventSourceTypeMismatch(string(apicommon.MinioEvent)),
		}, nil
	}

	var minioEventSource *apicommon.S3Artifact
	err := yaml.Unmarshal(eventSource.Value, &minioEventSource)
	if err != nil {
		listener.Logger.WithError(err).Error("failed to parse the minio event source")
		return &gateways.ValidEventSource{
			Reason:  err.Error(),
			IsValid: false,
		}, nil
	}

	if err := validate(minioEventSource); err != nil {
		return &gateways.ValidEventSource{
			Reason:  err.Error(),
			IsValid: false,
		}, nil
	}
	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func validate(eventSource *apicommon.S3Artifact) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.AccessKey == nil {
		return fmt.Errorf("access key can't be empty")
	}
	if eventSource.SecretKey == nil {
		return fmt.Errorf("secret key can't be empty")
	}
	if eventSource.Endpoint == "" {
		return fmt.Errorf("endpoint url can't be empty")
	}
	if eventSource.Bucket != nil && eventSource.Bucket.Name == "" {
		return fmt.Errorf("bucket name can't be empty")
	}
	if eventSource.Events != nil {
		for _, event := range eventSource.Events {
			if minio.NotificationEventType(event) == "" {
				return fmt.Errorf("unknown event %s", event)
			}
		}
	}
	return nil
}
