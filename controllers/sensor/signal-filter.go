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

package sensor

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	v1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
	"github.com/argoproj/argo-events/common"
)

// createEscalationEvent creates a k8 event for escalation
func (se *sensorExecutor) createEscalationEvent(policy *v1alpha1.EscalationPolicy, signalFilterName string) error {
	se.log.Info().Interface("policy", policy).Msg("printing policy")
	event := common.GetK8Event(&common.K8Event{
		Name: policy.Name,
		Namespace: se.sensor.Namespace,
		ReportingInstance: se.sensor.Name,
		ReportingController: se.sensor.Name,
		Labels: map[string]string{
			common.LabelEventSeen: "",
			common.LabelSignalName: signalFilterName,
		},
		Type: common.LabelArgoEventsEscalationKind,
		Action: string(policy.Level),
		Reason: policy.Message,
	})
	err := common.CreateK8Event(event, se.kubeClient)
	return err
}

// apply the signal filters to an event
func (se *sensorExecutor) filterEvent(f v1alpha1.SignalFilter, event *v1alpha.Event) (bool, error) {
	dataRes, err := se.filterData(f.Data.Filters, event)
	// generate sensor failure event and mark sensor as failed
	if err != nil {
		return false, err
	}
	if !dataRes {
		err = se.createEscalationEvent(f.Data.EscalationPolicy, f.Name)
		if err != nil {
			return false, err
		}
	}
	timeRes, err := se.filterTime(f.Time, &event.Context.EventTime)
	// generate sensor failure event and mark sensor as failed
	if err != nil {
		return false, err
	}
	if !timeRes {
		err = se.createEscalationEvent(f.Time.EscalationPolicy, f.Name)
		if err != nil {
			return false, err
		}
	}
	ctxRes := se.filterContext(f.Context, &event.Context)
	if !ctxRes {
		err = se.createEscalationEvent(f.Context.EscalationPolicy, f.Name)
		if err != nil {
			return false, err
		}
	}
	return timeRes && ctxRes && dataRes, err
}

// applyTimeFilter checks the eventTime against the timeFilter:
// 1. the eventTime is greater than or equal to the start time
// 2. the eventTime is less than the end time
// returns true if 1 and 2 are true and false otherwise
func (se *sensorExecutor) filterTime(timeFilter *v1alpha1.TimeFilter, eventTime *metav1.Time) (bool, error) {
	if timeFilter != nil {
		currentT := time.Now().UTC()
		se.log.Info().Str("current-time", currentT.String()).Msg("current time")
		currentTStr := fmt.Sprintf("%d-%s-%d", currentT.Year(), int(currentT.Month()), currentT.Day())

		if timeFilter.Start != "" && timeFilter.Stop != "" {
			se.log.Info().Str("start time format", currentTStr + " " + timeFilter.Start).Msg("start time format")
			startTime, err := time.Parse("2006-01-02 15:04:05", currentTStr + " " + timeFilter.Start)
			if err != nil {
				fmt.Println(err)
				return false, err
			}
			se.log.Info().Str("start time", startTime.String()).Msg("start time")
			startTime = startTime.UTC()
			se.log.Info().Str("stop time format", currentTStr + " " + timeFilter.Stop).Msg("stop time format")
			stopTime, err := time.Parse("2006-01-02 15:04:05", currentTStr + " " + timeFilter.Stop)
			if err != nil {
				fmt.Println(err)
				return false, err
			}
			se.log.Info().Str("stop time", stopTime.String()).Msg("stop time")
			stopTime = stopTime.UTC()
			return (startTime.Before(eventTime.Time) || stopTime.Equal(eventTime.Time)) && eventTime.Time.Before(stopTime), nil
		}
		if timeFilter.Start != "" {
			// stop is nil - does not have an end
			startTime, err := time.Parse("2006-01-02 15:04:05", currentTStr + " " + timeFilter.Start)
			if err != nil {
				return false, err
			}
			se.log.Info().Str("start time", startTime.String()).Msg("start time")
			startTime = startTime.UTC()
			return startTime.Before(eventTime.Time) || startTime.Equal(eventTime.Time), nil
		}
		if timeFilter.Stop != "" {
			stopTime, err := time.Parse("2016-01-02 15:04:05", currentTStr + " " + timeFilter.Stop)
			if err != nil {
				return false, err
			}
			se.log.Info().Str("stop time", stopTime.String()).Msg("stop time")
			stopTime = stopTime.UTC()
			return eventTime.Time.Before(stopTime), nil
		}
	}
	se.log.Info().Msg("NO time filter")
	return true, nil
}

// applyContextFilter checks the expected EventContext against the actual EventContext
// values are only enforced if they are non-zero values
// map types check that the expected map is a subset of the actual map
func (se *sensorExecutor) filterContext(expected *v1alpha.EventContext, actual *v1alpha.EventContext) bool {
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
func (se *sensorExecutor) filterData(dataFilters []*v1alpha1.DataFilter, event *v1alpha.Event) (bool, error) {
	// TODO: use the event.Context.SchemaURL to figure out correct data format to unmarshal to
	// for now, let's just use a simple map[string]interface{} for arbitrary data
	if dataFilters == nil {
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
	for _, f := range dataFilters {
		res := gjson.GetBytes(js, f.Path)
		if !res.Exists() {
			return false, nil
		}
		switch f.Type {
		case v1alpha1.JSONTypeBool:
			val, err := strconv.ParseBool(f.Value)
			if err != nil {
				return false, err
			}
			if val != res.Bool() {
				return false, nil
			}
		case v1alpha1.JSONTypeNumber:
			val, err := strconv.ParseFloat(f.Value, 64)
			if err != nil {
				return false, err
			}
			if val != res.Float() {
				return false, nil
			}
		case v1alpha1.JSONTypeString:
			if f.Value != res.Str {
				return false, nil
			}
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
