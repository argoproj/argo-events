package controller

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// apply the signal filters to an event
func filterEvent(f v1alpha1.SignalFilter, event *v1alpha1.Event) (bool, error) {
	dataRes, err := filterData(f.Data, event)
	return filterTime(f.Time, &event.Context.EventTime) && filterContext(f.Context, &event.Context) && dataRes, err
}

// applyTimeFilter checks the eventTime against the timeFilter:
// 1. the eventTime is greater than or equal to the start time
// 2. the eventTime is less than the end time
// returns true if 1 and 2 are true and false otherwise
func filterTime(timeFilter *v1alpha1.TimeFilter, eventTime *metav1.Time) bool {
	if timeFilter != nil && eventTime != nil {
		return (timeFilter.Start.Before(eventTime) || timeFilter.Start.Equal(eventTime)) && eventTime.Before(&timeFilter.Stop)
	}
	return true
}

// applyContextFilter checks the expected EventContext against the actual EventContext
// values are only enforced if they are non-zero values
// map types check that the expected map is a subset of the actual map
func filterContext(expected *v1alpha1.EventContext, actual *v1alpha1.EventContext) bool {
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

// various supported media types
// TODO: add support for XML
const (
	MediaTypeJSON string = "application/json"
	//MediaTypeXML  string = "application/xml"
	MediaTypeYAML string = "application/yaml"
)

// applyDataFilter runs the dataFilter against the event's data
// returns (true, nil) when data passes filters, false otherwise
// TODO: split this function up into smaller pieces
func filterData(dataFilters []*v1alpha1.DataFilter, event *v1alpha1.Event) (bool, error) {
	// TODO: use the event.Context.SchemaURL to figure out correct data format to unmarshal to
	// for now, let's just use a simple map[string]interface{} for arbitrary data
	if event == nil {
		return false, fmt.Errorf("nil event")
	}
	if event.Data == nil || len(event.Data) == 0 {
		return true, nil
	}
	raw := event.Data
	var data map[string]interface{}
	// contentType is formatted as: '{type}; charset="xxx"'
	contents := strings.Split(event.Context.ContentType, ";")
	if len(contents) < 1 {
		return false, fmt.Errorf("event context ContentType not found: %s", contents)
	}
	switch contents[0] {
	case MediaTypeJSON:
		if err := json.Unmarshal(raw, &data); err != nil {
			return false, err
		}
		/*
			case MediaTypeXML:
				if err := xml.Unmarshal(raw, &data); err != nil {
					return false, err
				}
		*/
	case MediaTypeYAML:
		if err := yaml.Unmarshal(raw, &data); err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("unsupported event content type: %s", event.Context.ContentType)
	}
	// now let's marshal the data back into json in order to do gjson processing
	json, err := json.Marshal(data)
	if err != nil {
		return false, err
	}
	for _, f := range dataFilters {
		res := gjson.Get(string(json), f.Path)
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
