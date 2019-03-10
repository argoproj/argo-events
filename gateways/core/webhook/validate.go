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

package webhook

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"net/http"
)

// ValidateEventSource validates webhook event source
func (ese *WebhookEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(es.Data, parseEventSource, validateWebhook)
}

func validateWebhook(config interface{}) error {
	w := config.(*gwcommon.Webhook)
	if w == nil {
		return gwcommon.ErrNilEventSource
	}

	switch w.Method {
	case http.MethodHead, http.MethodPut, http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodTrace:
	default:
		return fmt.Errorf("unknown HTTP method %s", w.Method)
	}

	return gwcommon.ValidateWebhook(w)
}
