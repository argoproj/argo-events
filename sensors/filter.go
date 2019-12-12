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

package sensors

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

// various supported media types
const (
	MediaTypeJSON string = "application/json"
)

func (sensorCtx *sensorContext) applyFilter(notification *notification) error {
	// apply filters if any.
	ok, err := sensorCtx.filterEvent(notification.eventDependency.Filters, notification.event)
	if err != nil {
		return err
	}
	if !ok {
		return errors.Errorf("failed to apply filter on event dependency %s", notification.eventDependency.Name)
	}
	return nil
}

// apply the filters to an event
func (sensorCtx *sensorContext) filterEvent(filter v1alpha1.EventDependencyFilter, event *cloudevents.Event) (bool, error) {
	dataFilter, err := sensorCtx.filterData(filter.Data, event)
	if err != nil {
		return false, err
	}
	timeFilter, err := sensorCtx.filterTime(filter.Time, event.Context.GetTime())
	if err != nil {
		return false, err
	}
	ctxFilter := sensorCtx.filterContext(filter.Context, event)

	return timeFilter && ctxFilter && dataFilter, err
}

// applyTimeFilter checks the eventTime against the timeFilter:
// 1. the eventTime is greater than or equal to the start time
// 2. the eventTime is less than the end time
// returns true if 1 and 2 are true and false otherwise
func (sensorCtx *sensorContext) filterTime(timeFilter *v1alpha1.TimeFilter, eventTime time.Time) (bool, error) {
	if timeFilter != nil {
		sensorCtx.logger.WithField(common.LabelTime, eventTime.String()).Info("event time")
		utc := time.Now().UTC()
		currentTime := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC).Format(common.StandardYYYYMMDDFormat)
		sensorCtx.logger.WithField(common.LabelTime, currentTime).Info("current time")

		if timeFilter.Start != "" && timeFilter.Stop != "" {
			startTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTime, timeFilter.Start))
			if err != nil {
				return false, err
			}
			sensorCtx.logger.WithField(common.LabelTime, startTime.String()).Info("start time")
			startTime = startTime.UTC()

			stopTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTime, timeFilter.Stop))
			if err != nil {
				return false, err
			}

			sensorCtx.logger.WithField(common.LabelTime, stopTime.String()).Info("stop time")
			stopTime = stopTime.UTC()

			return (startTime.Before(eventTime) || stopTime.Equal(eventTime)) && eventTime.Before(stopTime), nil
		}

		if timeFilter.Start != "" {
			// stop is nil - does not have an end
			startTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTime, timeFilter.Start))
			if err != nil {
				return false, err
			}

			sensorCtx.logger.WithField(common.LabelTime, startTime.String()).Info("start time")
			startTime = startTime.UTC()
			return startTime.Before(eventTime) || startTime.Equal(eventTime), nil
		}

		if timeFilter.Stop != "" {
			stopTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTime, timeFilter.Stop))
			if err != nil {
				return false, err
			}

			sensorCtx.logger.WithField(common.LabelTime, stopTime.String()).Info("stop time")
			stopTime = stopTime.UTC()
			return eventTime.Before(stopTime), nil
		}
	}
	return true, nil
}

// applyContextFilter checks the expected EventContext against the actual EventContext
// values are only enforced if they are non-zero values
// map types check that the expected map is a subset of the actual map
func (sensorCtx *sensorContext) filterContext(expected *cloudevents.Event, actual *cloudevents.Event) bool {
	if expected == nil {
		return true
	}
	if actual == nil {
		return false
	}
	res := true
	if expected.Type() != "" {
		res = res && expected.Type() == actual.Type()
	}
	if expected.SpecVersion() != "" {
		res = res && expected.SpecVersion() == actual.SpecVersion()
	}
	if expected.Source() != "" {
		res = res && reflect.DeepEqual(expected.Source(), actual.Source())
	}
	if expected.DataContentType() != "" {
		res = res && expected.DataContentType() == actual.DataContentType()
	}
	return res
}

// applyDataFilter runs the dataFilter against the event's data
// returns (true, nil) when data passes filters, false otherwise
// TODO: split this function up into smaller pieces
func (sensorCtx *sensorContext) filterData(data []v1alpha1.DataFilter, event *cloudevents.Event) (bool, error) {
	if data == nil {
		return true, nil
	}
	if event == nil {
		return false, fmt.Errorf("nil event")
	}
	payload, err := event.DataBytes()
	if err != nil {
		return false, err
	}
	if payload == nil || len(payload) == 0 {
		return true, nil
	}
	if event.DataContentType() != MediaTypeJSON {
		return false, fmt.Errorf("unsupported event content type: %s", event.DataContentType())
	}
	var js *json.RawMessage
	if err := json.Unmarshal(payload, &js); err != nil {
		return false, err
	}
	var jsData []byte
	jsData, err = json.Marshal(js)
	if err != nil {
		return false, err
	}
filter:
	for _, f := range data {
		res := gjson.GetBytes(jsData, f.Path)
		if !res.Exists() {
			return false, nil
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
				val, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return false, err
				}
				if val == res.Float() {
					continue filter
				}
			}
			return false, nil

		case v1alpha1.JSONTypeString:
			for _, value := range f.Value {
				exp, err := regexp.Compile(value)
				if err != nil {
					return false, err
				}

				if exp.Match([]byte(res.Str)) {
					continue filter
				}
			}
			return false, nil

		default:
			return false, fmt.Errorf("unsupported JSON type %s", f.Type)
		}
	}
	return true, nil
}
