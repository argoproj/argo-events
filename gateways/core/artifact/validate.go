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
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/minio/minio-go"
)

// ValidateEventSource validates a s3 event source
func (ese *S3EventSourceExecutor) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(eventSource, ArgoEventsEventSourceVersion, parseEventSource, validateArtifact)
}

// validates an artifact
func validateArtifact(config interface{}) error {
	a := config.(*apicommon.S3Artifact)
	if a == nil {
		return gwcommon.ErrNilEventSource
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
	if a.Events != nil {
		for _, event := range a.Events {
			if minio.NotificationEventType(event) == "" {
				return fmt.Errorf("unknown event %s", event)
			}
		}
	}
	return nil
}
