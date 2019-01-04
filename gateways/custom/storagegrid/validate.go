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

package storagegrid

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	"strings"
)

// ValidateEventSource validates gateway event source
func (ese *StorageGridEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	sg, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}
	if sg == nil {
		return v, gateways.ErrEmptyEventSource
	}
	if sg.Port == "" {
		return v, fmt.Errorf("%+v, must specify port", gateways.ErrInvalidEventSource)
	}
	if sg.Endpoint == "" {
		return v, fmt.Errorf("%+v, must specify endpoint", gateways.ErrInvalidEventSource)
	}
	if !strings.HasPrefix(sg.Endpoint, "/") {
		return v, fmt.Errorf("%+v, endpoint must start with '/'", gateways.ErrInvalidEventSource)
	}
	return v, nil
}
