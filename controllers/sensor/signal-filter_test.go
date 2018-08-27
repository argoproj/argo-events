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
	"testing"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_filterTime(t *testing.T) {
	type args struct {
		timeFilter *v1alpha1.TimeFilter
		eventTime  *metav1.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil filter and time",
			args: args{timeFilter: nil, eventTime: nil},
			want: true,
		},
		{
			name: "-∞ start and ∞ stop",
			args: args{
				timeFilter: &v1alpha1.TimeFilter{},
				eventTime: &metav1.Time{
					Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
			},
			want: true,
		},
		{
			name: "-∞ start and finite stop",
			args: args{timeFilter: &v1alpha1.TimeFilter{
				Stop: &metav1.Time{
					Time: time.Date(2018, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
			}, eventTime: &metav1.Time{
				Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
			}},
			want: true,
		},
		{
			name: "finite start and ∞ stop",
			args: args{timeFilter: &v1alpha1.TimeFilter{
				Start: &metav1.Time{
					Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
			}, eventTime: &metav1.Time{
				Time: time.Date(2017, time.May, 10, 0, 0, 0, 0, time.UTC),
			}},
			want: true,
		},
		{
			name: "event > start && event > stop",
			args: args{timeFilter: &v1alpha1.TimeFilter{
				Start: &metav1.Time{
					Time: time.Date(2012, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
				Stop: &metav1.Time{
					Time: time.Date(2015, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
			}, eventTime: &metav1.Time{
				Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
			}},
			want: false,
		},
		{
			name: "event > start && event == stop",
			args: args{timeFilter: &v1alpha1.TimeFilter{
				Start: &metav1.Time{
					Time: time.Date(2012, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
				Stop: &metav1.Time{
					Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
			}, eventTime: &metav1.Time{
				Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
			}},
			want: false,
		},
		{
			name: "event > start && event < stop",
			args: args{timeFilter: &v1alpha1.TimeFilter{
				Start: &metav1.Time{
					Time: time.Date(2012, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
				Stop: &metav1.Time{
					Time: time.Date(2017, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
			}, eventTime: &metav1.Time{
				Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
			}},
			want: true,
		},
		{
			name: "event == start && event < stop",
			args: args{timeFilter: &v1alpha1.TimeFilter{
				Start: &metav1.Time{
					Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
				Stop: &metav1.Time{
					Time: time.Date(2017, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
			}, eventTime: &metav1.Time{
				Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
			}},
			want: true,
		},
		{
			name: "event < start && event < stop",
			args: args{timeFilter: &v1alpha1.TimeFilter{
				Start: &metav1.Time{
					Time: time.Date(2017, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
				Stop: &metav1.Time{
					Time: time.Date(2018, time.May, 10, 0, 0, 0, 0, time.UTC),
				},
			}, eventTime: &metav1.Time{
				Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
			}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterTime(tt.args.timeFilter, tt.args.eventTime); got != tt.want {
				t.Errorf("filterTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filterContext(t *testing.T) {
	type args struct {
		expected *v1alpha1.EventContext
		actual   *v1alpha1.EventContext
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil expected",
			args: args{
				expected: nil,
				actual: &v1alpha1.EventContext{
					EventType: "argo.io.event",
				},
			},
			want: true,
		},
		{
			name: "nil actual, non-nil expected",
			args: args{
				expected: &v1alpha1.EventContext{
					EventType: "argo.io.event",
				},
				actual: nil,
			},
			want: false,
		},
		{
			name: "eventType",
			args: args{expected: &v1alpha1.EventContext{
				EventType: "argo.io.event",
			}, actual: &v1alpha1.EventContext{
				EventType: "argo.io.event",
			}},
			want: true,
		},
		{
			name: "eventTypeVersion",
			args: args{expected: &v1alpha1.EventContext{
				EventTypeVersion: "v1",
			}, actual: &v1alpha1.EventContext{
				EventTypeVersion: "v1",
			}},
			want: true,
		},
		{
			name: "cloudEventsVersion",
			args: args{expected: &v1alpha1.EventContext{
				CloudEventsVersion: "v1",
			}, actual: &v1alpha1.EventContext{
				CloudEventsVersion: "v1",
			}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterContext(tt.args.expected, tt.args.actual); got != tt.want {
				t.Errorf("filterContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filterData(t *testing.T) {
	type args struct {
		dataFilters []*v1alpha1.DataFilter
		event       *v1alpha1.Event
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "nil event",
			args:    args{dataFilters: nil, event: nil},
			want:    false,
			wantErr: true,
		},
		{
			name:    "unsupported content type",
			args:    args{dataFilters: nil, event: &v1alpha1.Event{Payload: []byte("a")}},
			want:    false,
			wantErr: true,
		},
		{
			name: "empty data",
			args: args{dataFilters: nil, event: &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					ContentType: "application/json",
				},
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "nil filters, JSON data",
			args: args{dataFilters: nil, event: &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					ContentType: "application/json",
				},
				Payload: []byte("{\"k\": \"v\"}"),
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter, JSON data",
			args: args{dataFilters: []*v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: "v",
				},
			}, event: &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					ContentType: "application/json",
				},
				Payload: []byte("{\"k\": \"v\"}"),
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "number filter, JSON data",
			args: args{dataFilters: []*v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeNumber,
					Value: "1.0",
				},
			}, event: &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					ContentType: "application/json",
				},
				Payload: []byte("{\"k\": \"1.0\"}"),
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "multiple filters, nested JSON data",
			args: args{dataFilters: []*v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeBool,
					Value: "true",
				},
				{
					Path:  "k1.k",
					Type:  v1alpha1.JSONTypeNumber,
					Value: "3.14",
				},
				{
					Path:  "k1.k2",
					Type:  v1alpha1.JSONTypeString,
					Value: "hello,world",
				},
			}, event: &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					ContentType: "application/json",
				},
				Payload: []byte("{\"k\": true, \"k1\": {\"k\": 3.14, \"k2\": \"hello, world\"}}"),
			}},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterData(tt.args.dataFilters, tt.args.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("filterData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapIsSubset(t *testing.T) {
	type args struct {
		sub map[string]string
		m   map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil sub, nil map",
			args: args{sub: nil, m: nil},
			want: true,
		},
		{
			name: "empty sub, empty map",
			args: args{sub: make(map[string]string), m: make(map[string]string)},
			want: true,
		},
		{
			name: "empty sub, non-empty map",
			args: args{sub: make(map[string]string), m: map[string]string{"k": "v"}},
			want: true,
		},
		{
			name: "disjoint",
			args: args{sub: map[string]string{"k1": "v1"}, m: map[string]string{"k": "v"}},
			want: false,
		},
		{
			name: "subset",
			args: args{sub: map[string]string{"k1": "v1"}, m: map[string]string{"k": "v", "k1": "v1"}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapIsSubset(tt.args.sub, tt.args.m); got != tt.want {
				t.Errorf("mapIsSubset() = %v, want %v", got, tt.want)
			}
		})
	}
}
