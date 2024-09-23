/*
Copyright 2018 The Argoproj Authors.

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

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/minio/minio-go/v7/pkg/notification"
)

// ValidateEventSource validates the minio event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.MinioEventSource)
}

func validate(eventSource *v1alpha1.S3Artifact) error {
	if eventSource == nil {
		return v1alpha1.ErrNilEventSource
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
			if notification.EventType(event) == "" {
				return fmt.Errorf("unknown event %s", event)
			}
		}
	}
	return nil
}
