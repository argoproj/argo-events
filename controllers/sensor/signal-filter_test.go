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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func Test_filterTime(t *testing.T) {
	timeFilter := &v1alpha1.TimeFilter{
		EscalationPolicy: &v1alpha1.EscalationPolicy{
			Name:    "time filter escalation",
			Message: "filter failed",
			Level:   v1alpha1.Alert,
		},
		Stop:  "17:14:00",
		Start: "10:11:00",
	}
	event := getCloudEvent()
	currentT := time.Now().UTC()
	currentMonth := fmt.Sprintf("%d", int(currentT.Month()))
	if int(currentT.Month()) < 10 {
		currentMonth = "0" + currentMonth
	}
	currentTStr := fmt.Sprintf("%d-%s-%d", currentT.Year(), currentMonth, currentT.Day())
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
		data  *v1alpha1.Data
		event *v1alpha1.Event
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
			args:    args{data: nil, event: &v1alpha1.Event{Payload: []byte("a")}},
			want:    true,
			wantErr: false,
		},
		{
			name: "empty data",
			args: args{data: nil, event: &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					ContentType: "application/json",
				},
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "nil filters, JSON data",
			args: args{data: nil, event: &v1alpha1.Event{
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
			args: args{
				data: &v1alpha1.Data{
					Filters: []*v1alpha1.DataFilter{
						{
							Path:  "k",
							Type:  v1alpha1.JSONTypeString,
							Value: "v",
						},
					},
				},
				event: &v1alpha1.Event{
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
			args: args{data: &v1alpha1.Data{
				Filters: []*v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeNumber,
						Value: "1.0",
					},
				},
			},
				event: &v1alpha1.Event{
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
			args: args{
				data: &v1alpha1.Data{
					Filters: []*v1alpha1.DataFilter{
						{
							Path:  "k",
							Type:  v1alpha1.JSONTypeString,
							Value: "v",
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
					},
				},
				event: &v1alpha1.Event{
					Context: v1alpha1.EventContext{
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
