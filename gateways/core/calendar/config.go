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
	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
)

// CalendarEventSourceExecutor implements Eventing
type CalendarEventSourceExecutor struct {
	Log zerolog.Logger
}

// calSchedule describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
// +k8s:openapi-gen=true
type calSchedule struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule"`

	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string `json:"interval"`

	// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
	// RDATE defines the list of DATE-TIME values for recurring events.
	// EXDATE defines the list of DATE-TIME exceptions for recurring events.
	// the combination of these rules and dates combine to form a set of date times.
	// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
	Recurrence []string `json:"recurrence,omitempty"`
}

func parseEventSource(eventSource string) (*calSchedule, error) {
	var c *calSchedule
	err := yaml.Unmarshal([]byte(eventSource), &c)
	if err != nil {
		return nil, err
	}
	return c, err
}
