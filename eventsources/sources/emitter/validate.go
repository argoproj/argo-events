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

package emitter

import (
	"context"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates emitter event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.EmitterEventSource)
}

func validate(eventSource *v1alpha1.EmitterEventSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.Broker == "" {
		return errors.New("broker url must be specified")
	}
	if eventSource.ChannelName == "" {
		return errors.New("channel name must be specified")
	}
	if eventSource.ChannelKey == "" {
		return errors.New("channel key secret selector must be specified")
	}
	if eventSource.TLS != nil {
		return v1alpha1.ValidateTLSConfig(eventSource.TLS)
	}
	return nil
}
