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
	"net/http"
	"strconv"
	"strings"

	"github.com/argoproj/argo-events/gateways"
)

// ValidateEventSource validates webhook event source
func (ese *WebhookEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	w, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}

	if w == nil {
		return v, fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidEventSource)
	}

	switch w.Method {
	case http.MethodHead, http.MethodPut, http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodTrace:
	default:
		return v, fmt.Errorf("%+v, unknown HTTP method %s", gateways.ErrInvalidEventSource, w.Method)
	}

	if w.Endpoint == "" {
		return v, fmt.Errorf("%+v, endpoint can't be empty", gateways.ErrInvalidEventSource)
	}
	if w.Port == "" {
		return v, fmt.Errorf("%+v, port can't be empty", gateways.ErrInvalidEventSource)
	}

	if !strings.HasPrefix(w.Endpoint, "/") {
		return v, fmt.Errorf("%+v, endpoint must start with '/'", gateways.ErrInvalidEventSource)
	}

	if w.Port != "" {
		_, err := strconv.Atoi(w.Port)
		if err != nil {
			return v, fmt.Errorf("%+v, failed to parse server port %s. err: %+v", gateways.ErrInvalidEventSource, w.Port, err)
		}
	}
	return v, nil
}
