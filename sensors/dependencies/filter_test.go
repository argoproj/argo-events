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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestFilterContext(t *testing.T) {
	tests := []struct {
		name            string
		expectedContext *v1alpha1.EventContext
		actualContext   *v1alpha1.EventContext
		result          bool
	}{
		{
			name: "different event contexts",
			expectedContext: &v1alpha1.EventContext{
				Type: "webhook",
			},
			actualContext: &v1alpha1.EventContext{
				Type:   "calendar",
				Source: "calendar-gateway",
				ID:     "1",
				Time: metav1.Time{
					Time: time.Now().UTC(),
				},
				DataContentType: common.MediaTypeJSON,
				Subject:         "example-1",
			},
			result: false,
		},
		{
			name: "contexts are same",
			expectedContext: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			actualContext: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Now().UTC(),
				},
				DataContentType: common.MediaTypeJSON,
				Subject:         "example-1",
			},
			result: true,
		},
		{
			name:            "actual event context is nil",
			expectedContext: &v1alpha1.EventContext{},
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
			args:    args{data: nil, event: &v1alpha1.Event{Data: []byte("a")}},
			want:    true,
			wantErr: false,
		},
		{
			name: "empty data",
			args: args{data: nil, event: &v1alpha1.Event{
				Context: &v1alpha1.EventContext{
					DataContentType: ("application/json"),
				},
			}},
			want:    true,
			wantErr: false,
		},
		{
			name: "nil filters, JSON data",
			args: args{data: nil, event: &v1alpha1.Event{
				Context: &v1alpha1.EventContext{
					DataContentType: ("application/json"),
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
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"v\"}"),
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter EqualTo, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeString,
						Value:      []string{"v"},
						Comparator: "=",
					},
				},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"v\"}"),
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter NotEqualTo, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeString,
						Value:      []string{"b"},
						Comparator: "!=",
					},
				},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
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
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"1.0\"}"),
				}},
			want:    true,
			wantErr: false,
		},
		{
			name: "comparator filter GreaterThan return true, JSON data",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:       "k",
					Type:       v1alpha1.JSONTypeNumber,
					Value:      []string{"1.0"},
					Comparator: ">",
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"2.0\"}"),
				}},
			want:    true,
			wantErr: false,
		},
		{
			name: "comparator filter LessThanOrEqualTo return false, JSON data",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:       "k",
					Type:       v1alpha1.JSONTypeNumber,
					Value:      []string{"1.0"},
					Comparator: "<=",
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"2.0\"}"),
				}},
			want:    false,
			wantErr: false,
		},
		{
			name: "comparator filter NotEqualTo, JSON data",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:       "k",
					Type:       v1alpha1.JSONTypeNumber,
					Value:      []string{"1.0"},
					Comparator: "!=",
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"1.0\"}"),
				}},
			want:    false,
			wantErr: false,
		},
		{
			name: "comparator filter EqualTo, JSON data",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:       "k",
					Type:       v1alpha1.JSONTypeNumber,
					Value:      []string{"5.0"},
					Comparator: "=",
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"5.0\"}"),
				}},
			want:    true,
			wantErr: false,
		},
		{
			name: "comparator filter empty, JSON data",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"10.0"},
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"10.0\"}"),
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
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": true, \"k1\": {\"k\": 3.14, \"k2\": \"hello, world\"}}"),
				}},
			want:    false,
			wantErr: false,
		},
		{
			name: "string filter Regex, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "[k,k1.a.#(k2==\"v2\").k2]",
						Type:       v1alpha1.JSONTypeString,
						Value:      []string{"\\bv\\b.*\\bv2\\b"},
						Comparator: "=",
					},
				},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"v\", \"k1\": {\"a\": [{\"k2\": \"v2\"}]}}"),
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter Regex2, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "[k,k1.a.#(k2==\"v2\").k2,,k1.a.#(k2==\"v3\").k2]",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"(\\bz\\b.*\\bv2\\b)|(\\bv\\b.*(\\bv2\\b.*\\bv3\\b))"},
					},
				},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"v\", \"k1\": {\"a\": [{\"k2\": \"v2\"}, {\"k2\": \"v3\"}]}}"),
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter base64, uppercase template",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:     "k",
					Type:     v1alpha1.JSONTypeString,
					Value:    []string{"HELLO WORLD"},
					Template: "{{ b64dec .Input | upper }}",
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"aGVsbG8gd29ybGQ=\"}"),
				}},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter base64 template",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:       "k",
					Type:       v1alpha1.JSONTypeNumber,
					Value:      []string{"3.13"},
					Comparator: ">",
					Template:   "{{ b64dec .Input }}",
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"My4xNA==\"}"), // 3.14
				}},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter base64 template, comparator not equal",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:       "k",
					Type:       v1alpha1.JSONTypeString,
					Value:      []string{"hello world"},
					Template:   "{{ b64dec .Input }}",
					Comparator: "!=",
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"aGVsbG8gd29ybGQ\"}"),
				}},
			want:    true,
			wantErr: false,
		},
		{
			name: "string filter base64 template, regex",
			args: args{data: []v1alpha1.DataFilter{
				{
					Path:     "k",
					Type:     v1alpha1.JSONTypeString,
					Value:    []string{"world$"},
					Template: "{{ b64dec .Input }}",
				},
			},
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: ("application/json"),
					},
					Data: []byte("{\"k\": \"aGVsbG8gd29ybGQ=\"}"),
				}},
			want:    true,
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
	now := time.Now().UTC()
	eventTimes := [6]time.Time{
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 4, 5, 6, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 8, 9, 10, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 12, 13, 14, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 16, 17, 18, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 20, 21, 22, 0, time.UTC),
	}

	time1 := eventTimes[2].Format("15:04:05")
	time2 := eventTimes[4].Format("15:04:05")

	tests := []struct {
		name       string
		timeFilter *v1alpha1.TimeFilter
		results    [6]bool
	}{
		{
			name:       "no filter",
			timeFilter: nil,
			results:    [6]bool{true, true, true, true, true, true},
			// With no filter, any event time should pass
		},
		{
			name: "start < stop",
			timeFilter: &v1alpha1.TimeFilter{
				Start: time1,
				Stop:  time2,
			},
			results: [6]bool{false, false, true, true, false, false},
			//                             ~~~~~~~~~~
			//                            [time1     , time2)
		},
		{
			name: "stop < start",
			timeFilter: &v1alpha1.TimeFilter{
				Start: time2,
				Stop:  time1,
			},
			results: [6]bool{true, true, false, false, true, true},
			//               ~~~~~~~~~~                ~~~~~~~~~~
			//              [          , time1)       [time2     , )
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for i, eventTime := range eventTimes {
				result, err := filterTime(test.timeFilter, eventTime)
				assert.Nil(t, err)
				assert.Equal(t, test.results[i], result)
			}
		})
	}
}

func TestFilterEvent(t *testing.T) {
	now := time.Now().UTC()
	eventTime := time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC)

	filter := v1alpha1.EventDependencyFilter{
		Time: &v1alpha1.TimeFilter{
			Start: "09:09:09",
			Stop:  "19:19:19",
		},
		Context: &v1alpha1.EventContext{
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
	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			Type:            "webhook",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			ID:              "1",
			Time:            metav1.Time{Time: eventTime},
			DataContentType: ("application/json"),
			Subject:         ("example-1"),
		},
		Data: []byte("{\"k\": \"v\"}"),
	}

	valid, err := filterEvent(&filter, event)
	assert.Nil(t, err)
	assert.Equal(t, valid, true)
}

func TestFilterExpr(t *testing.T) {
	tests := []struct {
		id             int
		event          *v1alpha1.Event
		filters        []v1alpha1.ExprFilter
		expectedResult bool
		expectedErrMsg string
	}{
		{
			id: 1,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": "b"}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `a == "b"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a",
							Name: "a",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 2,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": "c"}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `a != "b"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a",
							Name: "a",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 3,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "c"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "b"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
			},
			expectedResult: false,
			expectedErrMsg: "",
		},
		{
			id: 4,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": 2}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == 2`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 5,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": 2}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b < 1`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
			},
			expectedResult: false,
			expectedErrMsg: "",
		},
		{
			id: 6,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "start long string"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b =~ "start"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 7,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "long string"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b !~ "start"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 8,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "c"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "b"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
				{
					Expr: `b == "c"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 9,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": "y"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "b"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
				{
					Expr: `d == "d"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.d",
							Name: "d",
						},
					},
				},
			},
			expectedResult: false,
			expectedErrMsg: "",
		},
		{
			id: 10,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": "y"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `c == "c"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.c",
							Name: "c",
						},
					},
				},
				{
					Expr: `b == "b"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
			},
			expectedResult: false,
			expectedErrMsg: "path a.c does not exist",
		},
		{
			id: 11,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": {"e": true}}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `c == "c"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.c",
							Name: "c",
						},
					},
				},
				{
					Expr: `b == "b"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
				{
					Expr: `e == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.d.e",
							Name: "e",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 12,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "c": {"d": true}}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b != "b" && d == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
						{
							Path: "a.d",
							Name: "d",
						},
					},
				},
			},
			expectedResult: false,
			expectedErrMsg: "path a.d does not exist",
		},
		{
			id: 13,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "c": {"d": true}}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b != "b" && d == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
						{
							Path: "a.c.d",
							Name: "d",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 14,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "b", "c": {"d": false}}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "b" && d == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
						{
							Path: "a.d",
							Name: "d",
						},
					},
				},
			},
			expectedResult: false,
			expectedErrMsg: "path a.d does not exist",
		},
		{
			id: 15,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "b", "c": {"d": false}}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "b" || d == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
						{
							Path: "a.c.d",
							Name: "d",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			id: 16,
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "b", "c": {"d": false}, "e": 2}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "b" || (d == true && e == 2)`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
						{
							Path: "a.c.d",
							Name: "d",
						},
					},
				},
			},
			expectedResult: true,
			expectedErrMsg: "",
		},
	}

	for _, test := range tests {
		t.Logf("Run TestFilterExpr #%d", test.id)
		actualResult, actualErr := filterExpr(test.filters, test.event)

		if (test.expectedErrMsg != "" && actualErr == nil) ||
			(test.expectedErrMsg == "" && actualErr != nil) {
			t.Logf("Test #%d failed: expected error '%s' got '%v'",
				test.id, test.expectedErrMsg, actualErr)
		}
		if test.expectedErrMsg != "" {
			assert.EqualError(t, actualErr, test.expectedErrMsg)
		} else {
			assert.NoError(t, actualErr)
		}

		if test.expectedResult != actualResult {
			t.Logf("Test #%d failed: expected result '%t' got '%t'", test.id, test.expectedResult, actualResult)
		}
		assert.Equal(t, test.expectedResult, actualResult)
	}
}
