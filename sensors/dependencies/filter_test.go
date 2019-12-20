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
	"testing"
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilterContext(t *testing.T) {
	tests := []struct {
		name            string
		expectedContext *apicommon.EventContext
		actualContext   *apicommon.EventContext
		result          bool
	}{
		{
			name: "different event contexts",
			expectedContext: &apicommon.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
			},
			actualContext: &apicommon.EventContext{
				Type:            "calendar",
				SpecVersion:     "0.3",
				Source:          "calendar-gateway",
				ID:              "1",
				Time:            v1.MicroTime{Time: time.Now()},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			result: false,
		},
		{
			name: "contexts are same",
			expectedContext: &apicommon.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			actualContext: &apicommon.EventContext{
				Type:            "webhook",
				SpecVersion:     "0.3",
				Source:          "webhook-gateway",
				ID:              "1",
				Time:            v1.MicroTime{Time: time.Now()},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			result: true,
		},
		{
			name:            "actual event context is nil",
			expectedContext: &apicommon.EventContext{},
			actualContext:   nil,
			result:          false,
		},
		{
			name:            "expected event context is nil",
			expectedContext: nil,
			result:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := filterContext(test.expectedContext, test.actualContext)
			assert.Equal(t, test.result, result)
		})
	}
}

func TestFilterData(t *testing.T) {
	type args struct {
		data  []v1alpha1.DataFilter
		event *apicommon.Event
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "nil event",
			args:    args{data: nil, event: nil},
			want:    true,
			wantErr: false,
		},
		{
			name:    "unsupported content type",
			args:    args{data: nil, event: &apicommon.Event{Data: []byte("a")}},
			want:    true,
			wantErr: false,
		},
		{
			name: "empty data",
			args: args{data: nil, event: &apicommon.Event{
				Context: apicommon.EventContext{
					DataContentType: "application/json",
				},
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "nil filters, JSON data",
			args: args{data: nil, event: &apicommon.Event{
				Context: apicommon.EventContext{
					DataContentType: "application/json",
				},
				Data: []byte("{\"k\": \"v\"}"),
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"v"},
					},
				},
				event: &apicommon.Event{
					Context: apicommon.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte("{\"k\": \"v\"}"),
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "number filter, JSON data",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"1.0"},
				},
			},
				event: &apicommon.Event{
					Context: apicommon.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte("{\"k\": \"1.0\"}"),
				}},
			want:    true,
			wantErr: false,
		},
		{
			name: "multiple filters, nested JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"v"},
					},
					{
						Path:  "k1.k",
						Type:  v1alpha1.JSONTypeNumber,
						Value: []string{"3.14"},
					},
					{
						Path:  "k1.k2",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"hello,world", "hello there"},
					},
				},
				event: &apicommon.Event{
					Context: apicommon.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte("{\"k\": true, \"k1\": {\"k\": 3.14, \"k2\": \"hello, world\"}}"),
				}},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterData(tt.args.data, tt.args.event)
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

func TestFilterTime(t *testing.T) {
	currentT := time.Now().UTC()
	currentT = time.Date(currentT.Year(), currentT.Month(), currentT.Day(), 0, 0, 0, 0, time.UTC)
	currentTStr := currentT.Format(common.StandardYYYYMMDDFormat)
	eventTime, err := time.Parse(common.StandardTimeFormat, currentTStr+" 16:36:34")
	assert.Nil(t, err)

	tests := []struct {
		name       string
		timeFilter *v1alpha1.TimeFilter
		result     bool
	}{
		{
			name: "event time outside filter start and stop time",
			timeFilter: &v1alpha1.TimeFilter{
				Start: "09:09:09",
				Stop:  "09:10:09",
			},
			result: false,
		},
		{
			name: "filter with no stop time",
			timeFilter: &v1alpha1.TimeFilter{
				Start: "09:09:09",
				Stop:  "",
			},
			result: true,
		},
		{
			name: "filter with no start time",
			timeFilter: &v1alpha1.TimeFilter{
				Start: "09:09:09",
				Stop:  "17:09:09",
			},
			result: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := filterTime(test.timeFilter, eventTime)
			assert.Nil(t, err)
			assert.Equal(t, test.result, result)
		})
	}
}

func TestFilterEvent(t *testing.T) {
	currentT := time.Now().UTC()
	currentT = time.Date(currentT.Year(), currentT.Month(), currentT.Day(), 0, 0, 0, 0, time.UTC)
	currentTStr := currentT.Format(common.StandardYYYYMMDDFormat)
	eventTime, err := time.Parse(common.StandardTimeFormat, currentTStr+" 16:36:34")
	assert.Nil(t, err)

	filter := v1alpha1.EventDependencyFilter{
		Name: "test-filter",
		Time: &v1alpha1.TimeFilter{
			Start: "09:09:09",
			Stop:  "",
		},
		Context: &apicommon.EventContext{
			Type:   "webhook",
			Source: "webhook-gateway",
		},
		Data: []v1alpha1.DataFilter{
			{
				Path:  "k",
				Type:  v1alpha1.JSONTypeString,
				Value: []string{"v"},
			},
		},
	}
	event := &apicommon.Event{
		Context: apicommon.EventContext{
			Type:            "webhook",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			ID:              "1",
			Time:            v1.MicroTime{Time: eventTime},
			DataContentType: "application/json",
			Subject:         "example-1",
		},
		Data: []byte("{\"k\": \"v\"}"),
	}

	valid, err := filterEvent(&filter, event)
	assert.Nil(t, err)
	assert.Equal(t, valid, true)
}
