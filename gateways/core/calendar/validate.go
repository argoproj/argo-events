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
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

// Validate validates gateway configuration
func (ce *CalendarConfigExecutor) Validate(config *gateways.EventSourceContext) error {
	cal, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if cal == nil {
		return gateways.ErrEmptyConfig
	}
	if cal.Schedule == "" && cal.Interval == "" {
		return fmt.Errorf("%+v, must have either schedule or interval", gateways.ErrInvalidConfig)
	}
	_, err = resolveSchedule(cal)
	if err != nil {
		return err
	}
	return nil
}
