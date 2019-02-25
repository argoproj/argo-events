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

package common

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

var (
	ErrNilEventSource = fmt.Errorf("event source can't be nil")
)

func ValidateGatewayEventSource(eventSource string, parseEventSource func(string) (interface{}, error), validateEventSource func(interface{}) error) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	es, err := parseEventSource(eventSource)
	if err != nil {
		v.Reason = fmt.Sprintf("failed to parse event source. err: %+v", err)
		return v, nil
	}
	if err := validateEventSource(es); err != nil {
		v.Reason = fmt.Sprintf("failed to validate event source. err: %+v", err)
		return v, nil
	}
	v.IsValid = true
	return v, nil
}
