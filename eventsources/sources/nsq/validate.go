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
package nsq

import (
	"context"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates nsq event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.NSQEventSource)
}

func validate(eventSource *v1alpha1.NSQEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.HostAddress == "" {
		return errors.New("host address must be specified")
	}
	if eventSource.Topic == "" {
		return errors.New("topic must be specified")
	}
	if eventSource.Channel == "" {
		return errors.New("channel must be specified")
	}
	if eventSource.TLS != nil {
		return v1alpha1.ValidateTLSConfig(eventSource.TLS)
	}
	return nil
}
