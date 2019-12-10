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
	"strings"
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// various supported media types
// TODO: add support for XML
const (
	MediaTypeJSON string = "application/json"
	//MediaTypeXML  string = "application/xml"
	MediaTypeYAML string = "application/yaml"
)

// apply the eventDependency filters to an event
func (sec *sensorExecutionCtx) filterEvent(f v1alpha1.EventDependencyFilter, event *apicommon.Event) (bool, error) {
	dataRes, err := sec.filterData(f.Data, event)
	if err != nil {
		return false, err
	}
	timeRes, err := sec.filterTime(f.Time, &event.Context.EventTime)
	if err != nil {
		return false, err
	}
	ctxRes := sec.filterContext(f.Context, &event.Context)
	return timeRes && ctxRes && dataRes, err
}

// applyTimeFilter checks the eventTime against the timeFilter:
// 1. the eventTime is greater than or equal to the start time
// 2. the eventTime is less than the end time
// returns true if 1 and 2 are true and false otherwise
func (sec *sensorExecutionCtx) filterTime(timeFilter *v1alpha1.TimeFilter, eventTime *metav1.MicroTime) (bool, error) {
	if timeFilter != nil {
		sec.log.WithField(common.LabelTime, eventTime.String()).Info("event time")
		utc := time.Now().UTC()
		currentTime := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC).Format(common.StandardYYYYMMDDFormat)
		sec.log.WithField(common.LabelTime, currentTime).Info("current time")

		if timeFilter.Start != "" && timeFilter.Stop != "" {
			startTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTime, timeFilter.Start))
			if err != nil {
				return false, err
			}
			sec.log.WithField(common.LabelTime, startTime.String()).Info("start time")
			startTime = startTime.UTC()

			stopTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTime, timeFilter.Stop))
			if err != nil {
				return false, err
			}

			sec.log.WithField(common.LabelTime, stopTime.String()).Info("stop time")
			stopTime = stopTime.UTC()

			return (startTime.Before(eventTime.Time) || stopTime.Equal(eventTime.Time)) && eventTime.Time.Before(stopTime), nil
		}

		if timeFilter.Start != "" {
			// stop is nil - does not have an end
			startTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTime, timeFilter.Start))
			if err != nil {
				return false, err
			}

			sec.log.WithField(common.LabelTime, startTime.String()).Info("start time")
			startTime = startTime.UTC()
			return startTime.Before(eventTime.Time) || startTime.Equal(eventTime.Time), nil
		}

		if timeFilter.Stop != "" {
			stopTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTime, timeFilter.Stop))
			if err != nil {
				return false, err
			}

			sec.log.WithField(common.LabelTime, stopTime.String()).Info("stop time")
			stopTime = stopTime.UTC()
			return eventTime.Time.Before(stopTime), nil
		}
	}
	return true, nil
}

// applyContextFilter checks the expected EventContext against the actual EventContext
// values are only enforced if they are non-zero values
// map types check that the expected map is a subset of the actual map
func (sec *sensorExecutionCtx) filterContext(expected *apicommon.EventContext, actual *apicommon.EventContext) bool {
	if expected == nil {
		return true
	}
	if actual == nil {
		return false
	}
	res := true
	if expected.EventType != "" {
		res = res && expected.EventType == actual.EventType
	}
	if expected.EventTypeVersion != "" {
		res = res && expected.EventTypeVersion == actual.EventTypeVersion
	}
	if expected.CloudEventsVersion != "" {
		res = res && expected.CloudEventsVersion == actual.CloudEventsVersion
	}
	if expected.Source != nil {
		res = res && reflect.DeepEqual(expected.Source, actual.Source)
	}
	if expected.SchemaURL != nil {
		res = res && reflect.DeepEqual(expected.SchemaURL, actual.SchemaURL)
	}
	if expected.ContentType != "" {
		res = res && expected.ContentType == actual.ContentType
	}
	eExtensionRes := mapIsSubset(expected.Extensions, actual.Extensions)
	return res && eExtensionRes
}

// applyDataFilter runs the dataFilter against the event's data
// returns (true, nil) when data passes filters, false otherwise
// TODO: split this function up into smaller pieces
func (sec *sensorExecutionCtx) filterData(data []v1alpha1.DataFilter, event *apicommon.Event) (bool, error) {
	// TODO: use the event.Context.SchemaURL to figure out correct data format to unmarshal to
	// for now, let's just use a simple map[string]interface{} for arbitrary data
	if data == nil {
		return true, nil
	}
	if event == nil {
		return false, fmt.Errorf("nil event")
	}
	if event.Payload == nil || len(event.Payload) == 0 {
		return true, nil
	}
	js, err := renderEventDataAsJSON(event)
	if err != nil {
		return false, err
	}
filter:
	for _, f := range data {
		res := gjson.GetBytes(js, f.Path)
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

// checks that m contains the k,v pairs of sub
func mapIsSubset(sub map[string]string, m map[string]string) bool {
	for k, v := range sub {
		val, ok := m[k]
		if !ok {
			return false
		}
		if v != val {
			return false
		}
	}
	return true
}

func isJSON(b []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(b, &js) == nil
}

// util method to render an event's data as a JSON []byte
// json is a subset of yaml so this should work...
func renderEventDataAsJSON(e *apicommon.Event) ([]byte, error) {
	if e == nil {
		return nil, fmt.Errorf("event is nil")
	}
	raw := e.Payload
	// contentType is formatted as: '{type}; charset="xxx"'
	contents := strings.Split(e.Context.ContentType, ";")
	switch contents[0] {
	case MediaTypeJSON:
		if isJSON(raw) {
			return raw, nil
		}
		return nil, fmt.Errorf("event data is not valid JSON")
	case MediaTypeYAML:
		data, err := yaml.YAMLToJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("failed converting yaml event data to JSON: %s", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported event content type: %s", e.Context.ContentType)
	}
}
