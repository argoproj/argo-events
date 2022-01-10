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

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestFilterExpr(t *testing.T) {
	tests := []struct {
		name           string
		event          *v1alpha1.Event
		filters        []v1alpha1.ExprFilter
		operator       v1alpha1.LogicalOperator
		expectedResult bool
		expectedErrMsg string
	}{
		{
			name:  "nil event",
			event: nil,
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
			expectedResult: false,
			expectedErrMsg: "expr filter error (nil event)",
		},
		{
			name:  "unsupported content type",
			event: &v1alpha1.Event{Data: []byte("a")},
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
			expectedResult: false,
			expectedErrMsg: "expr filter error (event data not valid JSON)",
		},
		{
			name: "empty data",
			event: &v1alpha1.Event{
				Context: &v1alpha1.EventContext{
					DataContentType: "application/json",
				},
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
			name: "nil filters",
			event: &v1alpha1.Event{
				Context: &v1alpha1.EventContext{
					DataContentType: "application/json",
				},
			},
			filters:        nil,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "simple string equal",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "simple string different than",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "nested string equal",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: false,
			expectedErrMsg: "",
		},
		{
			name: "number equal",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "number less than",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: false,
			expectedErrMsg: "",
		},
		{
			name: "string contains",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "string does not contain",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "multiple filters, EMPTY operator",
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": "y"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "x"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
				{
					Expr: `d == "y"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.d",
							Name: "d",
						},
					},
				},
			},
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "multiple filters, AND operator",
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": "y"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "x"`,
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
			operator:       v1alpha1.AndLogicalOperator,
			expectedResult: false,
			expectedErrMsg: "",
		},
		{
			name: "multiple filters, OR operator",
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": "y"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "x"`,
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
			operator:       v1alpha1.OrLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "multiple filters, OR operator, one field not existing",
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
					Expr: `c == "c"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.c",
							Name: "c",
						},
					},
				},
			},
			operator:       v1alpha1.OrLogicalOperator,
			expectedResult: false,
			expectedErrMsg: "expr filter errors [path 'a.c' does not exist]",
		},
		{
			name: "multiple filters, OR operator, one field not existing but not reached because first filter is false",
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": {"e": true}}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "x"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
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
					Expr: `e == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.d.e",
							Name: "e",
						},
					},
				},
			},
			operator:       v1alpha1.OrLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "multiple filters, EMPTY operator, one field not existing but not reached because first filter is false",
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
					Expr: `c == "c"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.c",
							Name: "c",
						},
					},
				},
			},
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: false,
			expectedErrMsg: "",
		},
		{
			name: "multiple filters, EMPTY operator, one field not existing",
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": "y"}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "x"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
				{
					Expr: `c == "c"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.c",
							Name: "c",
						},
					},
				},
			},
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: false,
			expectedErrMsg: "expr filter error (path 'a.c' does not exist)",
		},
		{
			name: "multiple filters, AND operator, one field not existing",
			event: &v1alpha1.Event{
				Data: []byte(`{"a": {"b": "x", "d": {"e": true}}}`),
			},
			filters: []v1alpha1.ExprFilter{
				{
					Expr: `b == "x"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.b",
							Name: "b",
						},
					},
				},
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
					Expr: `e == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "a.d.e",
							Name: "e",
						},
					},
				},
			},
			operator:       v1alpha1.AndLogicalOperator,
			expectedResult: false,
			expectedErrMsg: "expr filter error (path 'a.c' does not exist)",
		},
		{
			name: "AND comparator inside expr (different than expr logical operator), one field not existing",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: false,
			expectedErrMsg: "expr filter error (path 'a.d' does not exist)",
		},
		{
			name: "AND comparator inside expr (different than expr logical operator)",
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
			operator:       v1alpha1.OrLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "OR comparator inside expr (different than expr logical operator)",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
		{
			name: "multiple comparators inside expr (different than expr logical operator)",
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
			operator:       v1alpha1.EmptyLogicalOperator,
			expectedResult: true,
			expectedErrMsg: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualResult, actualErr := filterExpr(test.filters, test.operator, test.event)

			if (test.expectedErrMsg != "" && actualErr == nil) ||
				(test.expectedErrMsg == "" && actualErr != nil) {
				t.Logf("'%s' test failed: expectedResult error '%s' got '%v'",
					test.name, test.expectedErrMsg, actualErr)
			}
			if test.expectedErrMsg != "" {
				assert.EqualError(t, actualErr, test.expectedErrMsg)
			} else {
				assert.NoError(t, actualErr)
			}

			if test.expectedResult != actualResult {
				t.Logf("'%s' test failed: expectedResult result '%t' got '%t'",
					test.name, test.expectedResult, actualResult)
			}
			assert.Equal(t, test.expectedResult, actualResult)
		})
	}
}
