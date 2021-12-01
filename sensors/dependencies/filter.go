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
	"strings"
	"text/template"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/Masterminds/sprig/v3"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
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

// filterEvent applies the filters to an Event
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

	exprFilter, err := filterExpr(filter.Exprs, event)
	if err != nil {
		return false, err
	}

	return timeFilter && ctxFilter && dataFilter && exprFilter, nil
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

// TODO review default result (false instead of true?)
// filterContext checks the expected EventContext against the actual EventContext
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

// filterData runs the dataFilter against the Event's data
// returns (true, nil) when data passes filters, false otherwise
func filterData(data []v1alpha1.DataFilter, event *v1alpha1.Event) (bool, error) {
	var err error

	if len(data) == 0 {
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
	err = json.Unmarshal(payload, &js)
	if err != nil {
		return false, err
	}
	var jsData []byte
	jsData, err = json.Marshal(js)
	if err != nil {
		return false, err
	}

	errMessages := make([]string, 0)
filterData:
	for _, f := range data {
		pathResult := gjson.GetBytes(jsData, f.Path)
		if !pathResult.Exists() {
			errMessages = append(errMessages, fmt.Sprintf("path '%s' does not exist", f.Path))
			continue
		}

		if f.Value == nil || len(f.Value) == 0 {
			errMessages = append(errMessages, "no values specified")
			continue
		}

		if f.Template != "" {
			tpl, tplErr := template.New("param").Funcs(sprig.HermeticTxtFuncMap()).Parse(f.Template)
			if tplErr != nil {
				errMessages = append(errMessages, tplErr.Error())
				continue
			}
			var buf bytes.Buffer
			execErr := tpl.Execute(&buf, map[string]interface{}{"Input": pathResult.String()})
			if execErr != nil {
				errMessages = append(errMessages, execErr.Error())
				continue
			}
			out := buf.String()
			if out == "" || out == "<no value>" {
				errMessages = append(errMessages, fmt.Sprintf("template evaluated to empty string or no value: '%s'", f.Template))
				continue
			}
			pathResult = gjson.Parse(strconv.Quote(out))
		}

		switch f.Type {

		case v1alpha1.JSONTypeBool:
			for _, value := range f.Value {
				val, boolErr := strconv.ParseBool(value)
				if boolErr != nil {
					errMessages = append(errMessages, boolErr.Error())
					continue filterData
				}
				if val == pathResult.Bool() {
					return true, nil
				}
			}
			continue filterData

		case v1alpha1.JSONTypeNumber:
			for _, value := range f.Value {
				filterVal, floatErr := strconv.ParseFloat(value, 64)
				eventVal := pathResult.Float()
				if floatErr != nil {
					errMessages = append(errMessages, floatErr.Error())
					continue filterData
				}

				switch f.Comparator {
				case v1alpha1.GreaterThanOrEqualTo:
					if eventVal >= filterVal {
						return true, nil
					}
				case v1alpha1.GreaterThan:
					if eventVal > filterVal {
						return true, nil
					}
				case v1alpha1.LessThan:
					if eventVal < filterVal {
						return true, nil
					}
				case v1alpha1.LessThanOrEqualTo:
					if eventVal <= filterVal {
						return true, nil
					}
				case v1alpha1.NotEqualTo:
					if eventVal != filterVal {
						return true, nil
					}
				case v1alpha1.EqualTo, v1alpha1.EmptyComparator:
					if eventVal == filterVal {
						return true, nil
					}
				}
			}
			continue filterData

		case v1alpha1.JSONTypeString:
			for _, value := range f.Value {
				exp, expErr := regexp.Compile(value)
				if expErr != nil {
					errMessages = append(errMessages, expErr.Error())
					continue filterData
				}

				match := exp.Match([]byte(pathResult.String()))
				switch f.Comparator {
				case v1alpha1.EqualTo, v1alpha1.EmptyComparator:
					if match {
						return true, nil
					}
				case v1alpha1.NotEqualTo:
					if !match {
						return true, nil
					}
				}
			}
			continue filterData

		default:
			errMessages = append(errMessages, fmt.Sprintf("unsupported JSON type '%s'", f.Type))
			continue filterData
		}
	}

	if len(errMessages) > 0 {
		return false, errors.New(strings.Join(errMessages, " / "))
	}

	return false, nil
}

// filterExpr applies expression based filters against event data
// expression evaluation is based on https://github.com/Knetic/govaluate
func filterExpr(filters []v1alpha1.ExprFilter, event *v1alpha1.Event) (bool, error) {
	var err error

	if len(filters) == 0 {
		return true, nil
	}
	if event == nil {
		return false, fmt.Errorf("nil event")
	}
	payload := event.Data
	if payload == nil {
		return true, nil
	}
	var js *json.RawMessage
	err = json.Unmarshal(payload, &js)
	if err != nil {
		return false, err
	}
	var jsData []byte
	jsData, err = json.Marshal(js)
	if err != nil {
		return false, err
	}

	errMessages := make([]string, 0)
filterExpr:
	for _, filter := range filters {
		parameters := map[string]interface{}{}
		for _, field := range filter.Fields {
			pathResult := gjson.GetBytes(jsData, field.Path)
			if !pathResult.Exists() {
				errMessages = append(errMessages, fmt.Sprintf("path '%s' does not exist", field.Path))
				continue filterExpr
			}
			parameters[field.Name] = pathResult.Value()
		}
		if len(parameters) == 0 {
			continue
		}
		expr, exprErr := govaluate.NewEvaluableExpression(filter.Expr)
		if exprErr != nil {
			errMessages = append(errMessages, exprErr.Error())
			continue
		}
		result, resErr := expr.Evaluate(parameters)
		if resErr != nil {
			errMessages = append(errMessages, resErr.Error())
			continue
		}
		if result == true {
			return true, nil
		}
	}

	if errMessages != nil && len(errMessages) > 0 {
		return false, errors.New(strings.Join(errMessages, " / "))
	}
	return false, nil
}
