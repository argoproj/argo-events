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
	"reflect"
	"testing"
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_filterTime(t *testing.T) {
	timeFilter := &v1alpha1.TimeFilter{
		Stop:  "17:14:00",
		Start: "10:11:00",
	}
	event := getCloudEvent()

	currentT := time.Now().UTC()
	currentT = time.Date(currentT.Year(), currentT.Month(), currentT.Day(), 0, 0, 0, 0, time.UTC)
	currentTStr := currentT.Format(common.StandardYYYYMMDDFormat)
	parsedTime, err := time.Parse(common.StandardTimeFormat, currentTStr+" 16:36:34")
	assert.Nil(t, err)
	event.Context.EventTime = metav1.MicroTime{
		Time: parsedTime,
	}
	sensor, err := getSensor()
	assert.Nil(t, err)
	sOptCtx := getsensorExecutionCtx(sensor)
	valid, err := sOptCtx.filterTime(timeFilter, &event.Context.EventTime)
	assert.Nil(t, err)
	assert.Equal(t, true, valid)

	// test invalid event
	timeFilter.Start = "09:09:09"
	timeFilter.Stop = "09:10:09"
	valid, err = sOptCtx.filterTime(timeFilter, &event.Context.EventTime)
	assert.Nil(t, err)
	assert.Equal(t, false, valid)

	// test no stop
	timeFilter.Start = "09:09:09"
	timeFilter.Stop = ""
	valid, err = sOptCtx.filterTime(timeFilter, &event.Context.EventTime)
	assert.Nil(t, err)
	assert.Equal(t, true, valid)

	// test no start
	timeFilter.Start = ""
	timeFilter.Stop = "17:09:09"
	valid, err = sOptCtx.filterTime(timeFilter, &event.Context.EventTime)
	assert.Nil(t, err)
	assert.Equal(t, true, valid)
}

func Test_filterContext(t *testing.T) {
	event := getCloudEvent()
	assert.NotNil(t, event)
	sensor, err := getSensor()
	assert.Nil(t, err)
	sOptCtx := getsensorExecutionCtx(sensor)
	assert.NotNil(t, sOptCtx)
	testCtx := event.Context.DeepCopy()
	valid := sOptCtx.filterContext(testCtx, &event.Context)
	assert.Equal(t, true, valid)
	testCtx.Source.Host = "dummy source"
	valid = sOptCtx.filterContext(testCtx, &event.Context)
	assert.Equal(t, false, valid)
}

func Test_filterData(t *testing.T) {
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
			args:    args{data: nil, event: &apicommon.Event{Payload: []byte("a")}},
			want:    true,
			wantErr: false,
		},
		{
			name: "empty data",
			args: args{data: nil, event: &apicommon.Event{
				Context: apicommon.EventContext{
					ContentType: "application/json",
				},
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "nil filters, JSON data",
			args: args{data: nil, event: &apicommon.Event{
				Context: apicommon.EventContext{
					ContentType: "application/json",
				},
				Payload: []byte("{\"k\": \"v\"}"),
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
						ContentType: "application/json",
					},
					Payload: []byte("{\"k\": \"v\"}"),
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
						ContentType: "application/json",
					},
					Payload: []byte("{\"k\": \"1.0\"}"),
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
						ContentType: "application/json",
					},
					Payload: []byte("{\"k\": true, \"k1\": {\"k\": 3.14, \"k2\": \"hello, world\"}}"),
				}},
			want:    false,
			wantErr: false,
		},
	}
	sensor, err := getSensor()
	assert.Nil(t, err)
	sOptCtx := getsensorExecutionCtx(sensor)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sOptCtx.filterData(tt.args.data, tt.args.event)
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

// this test is meant to cover the missing cases for those not covered in eventDependency-filter_test.go and trigger-params_test.go
func Test_renderEventDataAsJSON(t *testing.T) {
	type args struct {
		e *apicommon.Event
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "nil event",
			args:    args{e: nil},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing content type",
			args:    args{e: &apicommon.Event{}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid yaml content",
			args: args{e: &apicommon.Event{
				Context: apicommon.EventContext{
					ContentType: MediaTypeYAML,
				},
				Payload: []byte(`apiVersion: v1alpha1`),
			}},
			want:    []byte(`{"apiVersion":"v1alpha1"}`),
			wantErr: false,
		},
		{
			name: "json content marked as yaml",
			args: args{e: &apicommon.Event{
				Context: apicommon.EventContext{
					ContentType: MediaTypeYAML,
				},
				Payload: []byte(`{"apiVersion":5}`),
			}},
			want:    []byte(`{"apiVersion":5}`),
			wantErr: false,
		},
		{
			name: "invalid json content",
			args: args{e: &apicommon.Event{
				Context: apicommon.EventContext{
					ContentType: MediaTypeJSON,
				},
				Payload: []byte(`{5:"numberkey"}`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid yaml content",
			args: args{e: &apicommon.Event{
				Context: apicommon.EventContext{
					ContentType: MediaTypeYAML,
				},
				Payload: []byte(`%\x786`),
			}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderEventDataAsJSON(tt.args.e)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderEventDataAsJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderEventDataAsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
