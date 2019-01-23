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
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"

	"github.com/argoproj/argo-events/gateways"
	"github.com/minio/minio-go"
)

// ValidateEventSource validates a s3 event source
func (ese *S3EventSourceExecutor) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	a, err := parseEventSource(eventSource.Data)
	if err != nil {
		gateways.SetValidEventSource(v, fmt.Sprintf("%s. err: %s", gateways.ErrEventSourceParseFailed, err.Error()), false)
		return v, nil
	}
	if err = validateArtifact(a); err != nil {
		gateways.SetValidEventSource(v, err.Error(), false)
		return v, gateways.ErrInvalidEventSource
	}
	gateways.SetValidEventSource(v, "", true)
	return v, nil
}

// validates an artifact
func validateArtifact(a *apicommon.S3Artifact) error {
	if a == nil {
		return gateways.ErrEmptyEventSource
	}
	if a.AccessKey == nil {
		return fmt.Errorf("access key can't be empty")
	}
	if a.SecretKey == nil {
		return fmt.Errorf("secret key can't be empty")
	}
	if a.Endpoint == "" {
		return fmt.Errorf("endpoint url can't be empty")
	}
	if a.Bucket != nil && a.Bucket.Name == "" {
		return fmt.Errorf("bucket name can't be empty")
	}
	if a.Event != "" && minio.NotificationEventType(a.Event) == "" {
		return fmt.Errorf("unknown event %s", a.Event)
	}
	return nil
}
