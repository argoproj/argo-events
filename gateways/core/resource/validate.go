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

package resource

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

// ValidateEventSource validates gateway event source
func (rce *ResourceConfigExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	res, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}
	if res == nil {
		return v, fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidEventSource)
	}
	if res.Version == "" {
		return v, fmt.Errorf("%+v, resource version must be specified", gateways.ErrInvalidEventSource)
	}
	if res.Namespace == "" {
		return v, fmt.Errorf("%+v, resource namespace must be specified", gateways.ErrInvalidEventSource)
	}
	if res.Kind == "" {
		return v, fmt.Errorf("%+v, resource kind must be specified", gateways.ErrInvalidEventSource)
	}
	if res.Group == "" {
		return v, fmt.Errorf("%+v, resource group must be specified", gateways.ErrInvalidEventSource)
	}
	return v, nil
}
