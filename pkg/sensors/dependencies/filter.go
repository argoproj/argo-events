/*
Copyright 2018 The Argoproj Authors.

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
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Knetic/govaluate"
	sprig "github.com/Masterminds/sprig/v3"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
	"github.com/tidwall/gjson"
	lua "github.com/yuin/gopher-lua"
)

const (
	errMsgListSeparator = " / "
	errMsgTemplate      = "%s filter error (%s)"
	multiErrMsgTemplate = "%s filter errors [%s]"
)

// Filter filters the event with dependency's defined filters
func Filter(event *v1alpha1.Event, filter *v1alpha1.EventDependencyFilter, filtersLogicalOperator v1alpha1.LogicalOperator) (bool, error) {
	if filter == nil {
		return true, nil
	}

	ok, err := filterEvent(filter, filtersLogicalOperator, event)
	if err != nil {
		return false, err
	}

	return ok, nil
}

// filterEvent applies the filters to an Event
func filterEvent(filter *v1alpha1.EventDependencyFilter, operator v1alpha1.LogicalOperator, event *v1alpha1.Event) (bool, error) {
	var errMessages []string
	if operator == v1alpha1.OrLogicalOperator {
		errMessages = make([]string, 0)
	}

	exprFilter, exprErr := filterExpr(filter.Exprs, filter.ExprLogicalOperator, event)
	if exprErr != nil {
		if operator != v1alpha1.OrLogicalOperator {
			return false, exprErr
		}
		errMessages = append(errMessages, exprErr.Error())
	}

	dataFilter, dataErr := filterData(filter.Data, filter.DataLogicalOperator, event)
	if dataErr != nil {
		if operator != v1alpha1.OrLogicalOperator {
			return false, dataErr
		}
		errMessages = append(errMessages, dataErr.Error())
	}

	ctxFilter := filterContext(filter.Context, event.Context)

	timeFilter, timeErr := filterTime(filter.Time, event.Context.Time.Time)
	if timeErr != nil {
		if operator != v1alpha1.OrLogicalOperator {
			return false, timeErr
		}
		errMessages = append(errMessages, timeErr.Error())
	}

	scriptFilter, err := filterScript(filter.Script, event)
	if err != nil {
		return false, err
	}

	if operator == v1alpha1.OrLogicalOperator {
		pass := (filter.Exprs != nil && exprFilter) ||
			(filter.Data != nil && dataFilter) ||
			(filter.Context != nil && ctxFilter) ||
			(filter.Time != nil && timeFilter) ||
			(filter.Script != "" && scriptFilter)

		if len(errMessages) > 0 {
			return pass, errors.New(strings.Join(errMessages, errMsgListSeparator))
		}
		return pass, nil
	}
	return exprFilter && dataFilter && ctxFilter && timeFilter && scriptFilter, nil
}

// filterExpr applies expression based filters against event data
// expression evaluation is based on https://github.com/Knetic/govaluate
// in case "operator input" is equal to v1alpha1.OrLogicalOperator, filters are evaluated as mutual exclusive
func filterExpr(filters []v1alpha1.ExprFilter, operator v1alpha1.LogicalOperator, event *v1alpha1.Event) (bool, error) {
	if filters == nil {
		return true, nil
	}
	if event == nil {
		return false, fmt.Errorf(errMsgTemplate, "expr", "nil event")
	}
	payload := event.Data
	if payload == nil {
		return true, nil
	}
	if !gjson.Valid(string(payload)) {
		return false, fmt.Errorf(errMsgTemplate, "expr", "event data not valid JSON")
	}

	var errMessages []string
	if operator == v1alpha1.OrLogicalOperator {
		errMessages = make([]string, 0)
	}
filterExpr:
	for _, filter := range filters {
		parameters := map[string]interface{}{}
		for _, field := range filter.Fields {
			pathResult := gjson.GetBytes(payload, field.Path)
			if !pathResult.Exists() {
				errMsg := "path '%s' does not exist"
				if operator == v1alpha1.OrLogicalOperator {
					errMessages = append(errMessages, fmt.Sprintf(errMsg, field.Path))
					continue filterExpr
				} else {
					return false, fmt.Errorf(errMsgTemplate, "expr", fmt.Sprintf(errMsg, field.Path))
				}
			}
			parameters[field.Name] = pathResult.Value()
		}

		if len(parameters) == 0 {
			continue
		}

		expr, exprErr := govaluate.NewEvaluableExpression(filter.Expr)
		if exprErr != nil {
			if operator == v1alpha1.OrLogicalOperator {
				errMessages = append(errMessages, exprErr.Error())
				continue
			} else {
				return false, fmt.Errorf(errMsgTemplate, "expr", exprErr.Error())
			}
		}

		result, resErr := expr.Evaluate(parameters)
		if resErr != nil {
			if operator == v1alpha1.OrLogicalOperator {
				errMessages = append(errMessages, resErr.Error())
				continue
			} else {
				return false, fmt.Errorf(errMsgTemplate, "expr", resErr.Error())
			}
		}

		if result == true {
			if operator == v1alpha1.OrLogicalOperator {
				return true, nil
			}
		} else {
			if operator != v1alpha1.OrLogicalOperator {
				return false, nil
			}
		}
	}

	if operator == v1alpha1.OrLogicalOperator {
		if len(errMessages) > 0 {
			return false, fmt.Errorf(multiErrMsgTemplate, "expr", strings.Join(errMessages, errMsgListSeparator))
		}
		return false, nil
	} else {
		return true, nil
	}
}

// filterData runs the dataFilter against the Event's data
// returns (true, nil) when data passes filters, false otherwise
// in case "operator input" is equal to v1alpha1.OrLogicalOperator, filters are evaluated as mutual exclusive
func filterData(filters []v1alpha1.DataFilter, operator v1alpha1.LogicalOperator, event *v1alpha1.Event) (bool, error) {
	if len(filters) == 0 {
		return true, nil
	}
	if event == nil {
		return false, fmt.Errorf(errMsgTemplate, "data", "nil Event")
	}
	payload := event.Data
	if payload == nil {
		return true, nil
	}
	if !gjson.Valid(string(payload)) {
		return false, fmt.Errorf(errMsgTemplate, "data", "event data not valid JSON")
	}

	var errMessages []string
	if operator == v1alpha1.OrLogicalOperator {
		errMessages = make([]string, 0)
	}
filterData:
	for _, f := range filters {
		pathResult := gjson.GetBytes(payload, f.Path)
		if !pathResult.Exists() {
			errMsg := "path '%s' does not exist"
			if operator == v1alpha1.OrLogicalOperator {
				errMessages = append(errMessages, fmt.Sprintf(errMsg, f.Path))
				continue
			} else {
				return false, fmt.Errorf(errMsgTemplate, "data", fmt.Sprintf(errMsg, f.Path))
			}
		}

		if len(f.Value) == 0 {
			errMsg := "no values specified"
			if operator == v1alpha1.OrLogicalOperator {
				errMessages = append(errMessages, errMsg)
				continue
			} else {
				return false, fmt.Errorf(errMsgTemplate, "data", errMsg)
			}
		}

		if f.Template != "" {
			tpl, tplErr := template.New("param").Funcs(sprig.FuncMap()).Parse(f.Template)
			if tplErr != nil {
				if operator == v1alpha1.OrLogicalOperator {
					errMessages = append(errMessages, tplErr.Error())
					continue
				} else {
					return false, fmt.Errorf(errMsgTemplate, "data", tplErr.Error())
				}
			}

			var buf bytes.Buffer
			execErr := tpl.Execute(&buf, map[string]interface{}{
				"Input": pathResult.String(),
			})
			if execErr != nil {
				if operator == v1alpha1.OrLogicalOperator {
					errMessages = append(errMessages, execErr.Error())
					continue
				} else {
					return false, fmt.Errorf(errMsgTemplate, "data", execErr.Error())
				}
			}

			out := buf.String()
			if out == "" || out == "<no value>" {
				if operator == v1alpha1.OrLogicalOperator {
					errMessages = append(errMessages, fmt.Sprintf("template evaluated to empty string or no value: '%s'", f.Template))
					continue
				} else {
					return false, fmt.Errorf(errMsgTemplate, "data",
						fmt.Sprintf("template '%s' evaluated to empty string or no value", f.Template))
				}
			}

			pathResult = gjson.Parse(strconv.Quote(out))
		}

		switch f.Type {
		case v1alpha1.JSONTypeBool:
			for _, value := range f.Value {
				val, err := strconv.ParseBool(value)
				if err != nil {
					if operator == v1alpha1.OrLogicalOperator {
						errMessages = append(errMessages, err.Error())
						continue filterData
					} else {
						return false, fmt.Errorf(errMsgTemplate, "data", err.Error())
					}
				}

				if val == pathResult.Bool() {
					if operator == v1alpha1.OrLogicalOperator {
						return true, nil
					} else {
						continue filterData
					}
				}
			}

			if operator == v1alpha1.OrLogicalOperator {
				continue filterData
			} else {
				return false, nil
			}

		case v1alpha1.JSONTypeNumber:
			for _, value := range f.Value {
				filterVal, err := strconv.ParseFloat(value, 64)
				eventVal := pathResult.Float()
				if err != nil {
					if operator == v1alpha1.OrLogicalOperator {
						errMessages = append(errMessages, err.Error())
						continue filterData
					} else {
						return false, fmt.Errorf(errMsgTemplate, "data", err.Error())
					}
				}

				compareResult := false
				switch f.Comparator {
				case v1alpha1.GreaterThanOrEqualTo:
					if eventVal >= filterVal {
						compareResult = true
					}
				case v1alpha1.GreaterThan:
					if eventVal > filterVal {
						compareResult = true
					}
				case v1alpha1.LessThan:
					if eventVal < filterVal {
						compareResult = true
					}
				case v1alpha1.LessThanOrEqualTo:
					if eventVal <= filterVal {
						compareResult = true
					}
				case v1alpha1.NotEqualTo:
					if eventVal != filterVal {
						compareResult = true
					}
				case v1alpha1.EqualTo, v1alpha1.EmptyComparator:
					if eventVal == filterVal {
						compareResult = true
					}
				}

				if compareResult {
					if operator == v1alpha1.OrLogicalOperator {
						return true, nil
					} else {
						continue filterData
					}
				}
			}
			if operator == v1alpha1.OrLogicalOperator {
				continue filterData
			} else {
				return false, nil
			}

		case v1alpha1.JSONTypeString:
			for _, value := range f.Value {
				exp, err := regexp.Compile(value)
				if err != nil {
					if operator == v1alpha1.OrLogicalOperator {
						errMessages = append(errMessages, err.Error())
						continue filterData
					} else {
						return false, fmt.Errorf(errMsgTemplate, "data", err.Error())
					}
				}

				matchResult := false
				match := exp.Match([]byte(pathResult.String()))
				switch f.Comparator {
				case v1alpha1.EqualTo, v1alpha1.EmptyComparator:
					if match {
						matchResult = true
					}
				case v1alpha1.NotEqualTo:
					if !match {
						matchResult = true
					}
				}

				if matchResult {
					if operator == v1alpha1.OrLogicalOperator {
						return true, nil
					} else {
						continue filterData
					}
				}
			}

			if operator == v1alpha1.OrLogicalOperator {
				continue filterData
			} else {
				return false, nil
			}

		default:
			errMsg := "unsupported JSON type '%s'"
			if operator == v1alpha1.OrLogicalOperator {
				errMessages = append(errMessages, fmt.Sprintf(errMsg, f.Type))
				continue filterData
			} else {
				return false, fmt.Errorf(errMsgTemplate, "data", fmt.Sprintf(errMsg, f.Type))
			}
		}
	}

	if operator == v1alpha1.OrLogicalOperator {
		if len(errMessages) > 0 {
			return false, fmt.Errorf(multiErrMsgTemplate, "data", strings.Join(errMessages, errMsgListSeparator))
		}
		return false, nil
	} else {
		return true, nil
	}
}

// filterContext checks the expectedResult EventContext against the actual EventContext
// values are only enforced if they are non-zero values
// map types check that the expectedResult map is a subset of the actual map
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

// filterTime checks the eventTime falls into time range specified by the timeFilter.
// Start is inclusive, and Stop is exclusive.
//
// if Start < Stop: eventTime must be in [Start, Stop)
//
//	0:00        Start       Stop        0:00
//	├───────────●───────────○───────────┤
//	            └─── OK ────┘
//
// if Stop < Start: eventTime must be in [Start, Stop@Next day)
//
// this is equivalent to: eventTime must be in [0:00, Stop) or [Start, 0:00@Next day)
//
//	0:00                    Start       0:00       Stop                     0:00
//	├───────────○───────────●───────────┼───────────○───────────●───────────┤
//	                        └───────── OK ──────────┘
//
//	0:00        Stop        Start       0:00
//	●───────────○───────────●───────────○
//	└─── OK ────┘           └─── OK ────┘
func filterTime(timeFilter *v1alpha1.TimeFilter, eventTime time.Time) (bool, error) {
	if timeFilter == nil {
		return true, nil
	}

	// Parse start and stop with timezone support
	var startTime, stopTime time.Time
	var startErr, stopErr error

	if timeFilter.Timezone != "" {
		// Use timezone-aware parsing
		startTime, startErr = sharedutil.ParseTimeInTimezone(timeFilter.Start, eventTime, timeFilter.Timezone)
		if startErr != nil {
			return false, fmt.Errorf(errMsgTemplate, "time", startErr.Error())
		}
		stopTime, stopErr = sharedutil.ParseTimeInTimezone(timeFilter.Stop, eventTime, timeFilter.Timezone)
		if stopErr != nil {
			return false, fmt.Errorf(errMsgTemplate, "time", stopErr.Error())
		}
	} else {
		// Default to UTC (backward compatible)
		startTime, startErr = sharedutil.ParseTime(timeFilter.Start, eventTime)
		if startErr != nil {
			return false, fmt.Errorf(errMsgTemplate, "time", startErr.Error())
		}
		stopTime, stopErr = sharedutil.ParseTime(timeFilter.Stop, eventTime)
		if stopErr != nil {
			return false, fmt.Errorf(errMsgTemplate, "time", stopErr.Error())
		}
	}

	// Filtering logic
	if startTime.Before(stopTime) {
		return (eventTime.After(startTime) || eventTime.Equal(startTime)) && eventTime.Before(stopTime), nil
	} else {
		return (eventTime.After(startTime) || eventTime.Equal(startTime)) || eventTime.Before(stopTime), nil
	}
}

func filterScript(script string, event *v1alpha1.Event) (bool, error) {
	if script == "" {
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
	if err := json.Unmarshal(payload, &js); err != nil {
		return false, err
	}
	var jsData []byte
	jsData, err := json.Marshal(js)
	if err != nil {
		return false, err
	}
	l := lua.NewState()
	defer l.Close()
	var payloadJson map[string]interface{}
	if err = json.Unmarshal(jsData, &payloadJson); err != nil {
		return false, err
	}
	lEvent := mapToTable(payloadJson)
	l.SetGlobal("event", lEvent)
	if err = l.DoString(script); err != nil {
		return false, err
	}
	lv := l.Get(-1)
	return lv == lua.LTrue, nil
}
