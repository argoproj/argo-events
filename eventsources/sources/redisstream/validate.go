/*
Copyright 2020 BlackRock, Inc.

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
package redisstream

import (
	"context"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates nats event source
func (el *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&el.EventSource)
}

func validate(eventSource *v1alpha1.RedisStreamEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.HostAddress == "" {
		return errors.New("host address must be specified")
	}
	if eventSource.Streams == nil {
		return errors.New("stream/streams must be specified")
	}
	if eventSource.Password != nil && eventSource.Namespace == "" {
		return errors.New("namespace must be defined in order to retrieve the password from the secret")
	}
	if eventSource.TLS != nil {
		return apicommon.ValidateTLSConfig(eventSource.TLS)
	}
	return nil
}
