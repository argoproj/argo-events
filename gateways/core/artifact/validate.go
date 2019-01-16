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
	"context"
	"fmt"

	"github.com/argoproj/argo-events/gateways"
	"github.com/minio/minio-go"
)

// ValidateEventSource validates a s3 event source
func (ese *S3EventSourceExecutor) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	artifact, err := parseEventSource(eventSource.Data)
	if err != nil {
		gateways.SetValidEventSource(v, fmt.Sprintf("%s. err: %s", gateways.ErrEventSourceParseFailed, err.Error()), false)
		return v, nil
	}
	if err = validateArtifact(artifact); err != nil {
		gateways.SetValidEventSource(v, err.Error(), false)
		return v, gateways.ErrInvalidEventSource
	}
	gateways.SetValidEventSource(v, "", true)
	return v, nil
}

// validates an artifact
func validateArtifact(artifact *s3Artifact) error {
	if artifact == nil {
		return gateways.ErrEmptyEventSource
	}
	if artifact.S3EventConfig == nil {
		return fmt.Errorf("s3 bucket configuration can't be empty")
	}
	if artifact.AccessKey == nil {
		return fmt.Errorf("access key can't be empty")
	}
	if artifact.SecretKey == nil {
		return fmt.Errorf("secret key can't be empty")
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
