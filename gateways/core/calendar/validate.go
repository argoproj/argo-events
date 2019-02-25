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

package calendar

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates gateway event source
func (ese *CalendarEventSourceExecutor) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(eventSource.Data, parseEventSource, validateSchedule)
}

func validateSchedule(config interface{}) error {
	cal := config.(*calSchedule)
	if cal == nil {
		return gwcommon.ErrNilEventSource
	}
	if cal.Schedule == "" && cal.Interval == "" {
		return fmt.Errorf("must have either schedule or interval")
	}
	if _, err := resolveSchedule(cal); err != nil {
		return err
	}
	return nil
}
