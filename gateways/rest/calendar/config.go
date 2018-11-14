package calendar

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
)

// CalendarConfigExecutor implements ConfigExecutor interface
type CalendarConfigExecutor struct {
	*gateways.HttpGatewayServerConfig
}

// calSchedule describes a time based dependency. One of the fields (schedule, interval, or recurrence) must be passed.
// Schedule takes precedence over interval; interval takes precedence over recurrence
type calSchedule struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	Schedule string

	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	Interval string

	// List of RRULE, RDATE and EXDATE lines for a recurring event, as specified in RFC5545.
	// RRULE is a recurrence rule which defines a repeating pattern for recurring events.
	// RDATE defines the list of DATE-TIME values for recurring events.
	// EXDATE defines the list of DATE-TIME exceptions for recurring events.
	// the combination of these rules and dates combine to form a set of date times.
	// NOTE: functionality currently only supports EXDATEs, but in the future could be expanded.
	Recurrence []string
}

func parseConfig(config string) (*calSchedule, error) {
	var c *calSchedule
	err := yaml.Unmarshal([]byte(config), &c)
	if err != nil {
		return nil, err
	}
	return c, err
}
