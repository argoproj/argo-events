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

package dependencies

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"text/template"
	"time"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/tidwall/gjson"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// Filter filters the event with dependency's defined filters
func Filter(event *v1alpha1.Event, filters *v1alpha1.EventDependencyFilter) (bool, error) {
	if filters == nil {
		return true, nil
	}
	ok, err := filterEvent(filters, event)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// apply the filters to an Event
func filterEvent(filter *v1alpha1.EventDependencyFilter, event *v1alpha1.Event) (bool, error) {
	dataFilter, err := filterData(filter.Data, event)
	if err != nil {
		return false, err
	}
	timeFilter, err := filterTime(filter.Time, event.Context.Time.Time)
	if err != nil {
		return false, err
	}
	ctxFilter := filterContext(filter.Context, event.Context)

	return timeFilter && ctxFilter && dataFilter, err
}

// filterTime checks the eventTime falls into time range specified by the timeFilter.
// Start is inclusive, and Stop is exclusive.
//
// if Start < Stop: eventTime must be in [Start, Stop)
//
//   0:00        Start       Stop        0:00
//   ├───────────●───────────○───────────┤
//               └─── OK ────┘
//
// if Stop < Start: eventTime must be in [Start, Stop@Next day)
//
// this is equivalent to: eventTime must be in [0:00, Stop) or [Start, 0:00@Next day)
//
//   0:00                    Start       0:00       Stop                     0:00
//   ├───────────○───────────●───────────┼───────────○───────────●───────────┤
//                           └───────── OK ──────────┘
//
//   0:00        Stop        Start       0:00
//   ●───────────○───────────●───────────○
//   └─── OK ────┘           └─── OK ────┘
func filterTime(timeFilter *v1alpha1.TimeFilter, eventTime time.Time) (bool, error) {
	if timeFilter == nil {
		return true, nil
	}

	// Parse start and stop
	startTime, err := common.ParseTime(timeFilter.Start, eventTime)
	if err != nil {
		return false, err
	}
	stopTime, err := common.ParseTime(timeFilter.Stop, eventTime)
	if err != nil {
		return false, err
	}

	// Filtering logic
	if startTime.Before(stopTime) {
		return (eventTime.After(startTime) || eventTime.Equal(startTime)) && eventTime.Before(stopTime), nil
	} else {
		return (eventTime.After(startTime) || eventTime.Equal(startTime)) || eventTime.Before(stopTime), nil
	}
}

// applyContextFilter checks the expected EventContext against the actual EventContext
// values are only enforced if they are non-zero values
// map types check that the expected map is a subset of the actual map
func filterContext(expected *v1alpha1.EventContext, actual *v1alpha1.EventContext) bool {
	if expected == nil {
		return true
	}
	if actual == nil {
		return false
	}
	res := true
	if expected.Type != "" {
		res = res && expected.Type == actual.Type
	}
	if expected.Subject != "" {
		res = res && expected.Subject == actual.Subject
	}
	if expected.Source != "" {
		res = res && expected.Source == actual.Source
	}
	if expected.DataContentType != "" {
		res = res && expected.DataContentType == actual.DataContentType
	}
	return res
}

// applyDataFilter runs the dataFilter against the Event's data
// returns (true, nil) when data passes filters, false otherwise
func filterData(data []v1alpha1.DataFilter, event *v1alpha1.Event) (bool, error) {
	if data == nil {
		return true, nil
	}
	if event == nil {
		return false, fmt.Errorf("nil Event")
	}
	payload := event.Data
	if payload == nil {
		return true, nil
	}
	var js *json.RawMessage
	if err := json.Unmarshal(payload, &js); err != nil {
		return false, err
	}
	var jsData []byte
	jsData, err := json.Marshal(js)
	if err != nil {
		return false, err
	}
filter:
	for _, f := range data {
		res := gjson.GetBytes(jsData, f.Path)
		if !res.Exists() {
			return false, nil
		}

		if f.Template != "" {
			tpl, err := template.New("param").Funcs(sprig.HermeticTxtFuncMap()).Parse(f.Template)
			if err != nil {
				return false, err
			}
			var buf bytes.Buffer
			if err := tpl.Execute(&buf, map[string]interface{}{
				"Input": res.String(),
			}); err != nil {
				return false, err
			}
			out := buf.String()
			if out == "" || out == "<no value>" {
				return false, fmt.Errorf("template evaluated to empty string or no value: %s", f.Template)
			}
			res = gjson.Parse(strconv.Quote(out))
		}

		switch f.Type {
		case v1alpha1.JSONTypeBool:
			for _, value := range f.Value {
				val, err := strconv.ParseBool(value)
				if err != nil {
					return false, err
				}
				if val == res.Bool() {
					continue filter
				}
			}
			return false, nil

		case v1alpha1.JSONTypeNumber:
			for _, value := range f.Value {
				filterVal, err := strconv.ParseFloat(value, 64)
				eventVal := res.Float()
				if err != nil {
					return false, err
				}

				switch f.Comparator {
				case v1alpha1.GreaterThanOrEqualTo:
					if eventVal >= filterVal {
						continue filter
					}
				case v1alpha1.GreaterThan:
					if eventVal > filterVal {
						continue filter
					}
				case v1alpha1.LessThan:
					if eventVal < filterVal {
						continue filter
					}
				case v1alpha1.LessThanOrEqualTo:
					if eventVal <= filterVal {
						continue filter
					}
				case v1alpha1.NotEqualTo:
					if eventVal != filterVal {
						continue filter
					}
				case v1alpha1.EqualTo, v1alpha1.EmptyComparator:
					if eventVal == filterVal {
						continue filter
					}
				}
			}
			return false, nil

		case v1alpha1.JSONTypeString:
			for _, value := range f.Value {
				exp, err := regexp.Compile(value)
				if err != nil {
					return false, err
				}

				match := exp.Match([]byte(res.String()))
				switch f.Comparator {
				case v1alpha1.EqualTo, v1alpha1.EmptyComparator:
					if match {
						continue filter
					}
				case v1alpha1.NotEqualTo:
					if !match {
						continue filter
					}
				}
			}
			return false, nil

		default:
			return false, fmt.Errorf("unsupported JSON type %s", f.Type)
		}
	}
	return true, nil
}
