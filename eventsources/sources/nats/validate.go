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

package nats

import (
	"context"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates nats event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.NATSEventSource)
}

func validate(eventSource *v1alpha1.NATSEventsSource) error {
	if eventSource == nil {
		return common.ErrNilEventSource
	}
	if eventSource.URL == "" {
		return errors.New("url must be specified")
	}
	if eventSource.Subject == "" {
		return errors.New("subject must be specified")
	}
	if eventSource.TLS != nil {
		return apicommon.ValidateTLSConfig(eventSource.TLS)
	}
	switch eventSource.Auth {
	case v1alpha1.NATSAuthBasic:
		if eventSource.Username == nil || eventSource.Password == nil {
			return errors.New("Username and Password secrets must be specified")
		}
	case v1alpha1.NATSAuthToken:
		if eventSource.Token == nil {
			return errors.New("Token secret must be specified")
		}
	case v1alpha1.NATSAuthNKEY:
		if eventSource.NKey == nil {
			return errors.New("NKey secret must be specified")
		}
	case v1alpha1.NATSAuthCredential:
		if eventSource.Credential == nil {
			return errors.New("Credential secret must be specified")
		}
	}
	return nil
}
