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

package trello

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates a s3 event source
func (ese *TrelloEventSourceExecutor) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(eventSource.Data, parseEventSource, validateTrello)
}

// validates trello config
func validateTrello(config interface{}) error {
	tl := config.(*trello)
	if tl == nil {
		return gwcommon.ErrNilEventSource
	}
	if tl.URL == "" {
		return fmt.Errorf("url is not specified")
	}
	if tl.Token == nil {
		return fmt.Errorf("token can't be empty")
	}
	if tl.ApiKey == nil {
		return fmt.Errorf("api key can't be empty")
	}
	return gwcommon.ValidateWebhook(tl.Endpoint, tl.Port)
}
