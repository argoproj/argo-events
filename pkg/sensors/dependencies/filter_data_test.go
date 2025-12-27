package dependencies

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestFilterData(t *testing.T) {
	type args struct {
		data     []v1alpha1.DataFilter
		operator v1alpha1.LogicalOperator
		event    *v1alpha1.Event
	}

	tests := []struct {
		name           string
		args           args
		expectedResult bool
		expectErr      bool
	}{
		{
			name: "nil event",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"v"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event:    nil,
			},
			expectedResult: false,
			expectErr:      true,
		},
		{
			name: "unsupported content type",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"v"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event:    &v1alpha1.Event{Data: []byte("a")},
			},
			expectedResult: false,
			expectErr:      true,
		},
		{
			name: "empty data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"v"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
				},
			},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "nil filters, JSON data",
			args: args{
				data:     nil,
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v"}`),
				},
			},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "invalid filter path, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path: "",
						Type: v1alpha1.JSONTypeString,
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v"}`),
				},
			},
			expectedResult: false,
			expectErr:      true,
		},
		{
			name: "invalid filter type, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path: "k",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v"}`),
				},
			},
			expectedResult: false,
			expectErr:      true,
		},
		{
			name: "invalid filter values, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v"}`),
				},
			},
			expectedResult: false,
			expectErr:      true,
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
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v"}`),
				},
			},
			expectedResult: true,
			expectErr:      false,
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
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v"}`),
				},
			},
			expectedResult: true,
			expectErr:      false,
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
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v"}`),
				},
			},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "number filter (data: string, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeNumber,
						Value: []string{"1.0"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "1.0"}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "comparator filter GreaterThan return true (data: string, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"1.0"},
						Comparator: ">",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "2.0"}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "comparator filter LessThanOrEqualTo return false (data: string, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"1.0"},
						Comparator: "<=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "2.0"}`),
				}},
			expectedResult: false,
			expectErr:      false,
		},
		{
			name: "comparator filter NotEqualTo (data: string, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"1.0"},
						Comparator: "!=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "1.0"}`),
				}},
			expectedResult: false,
			expectErr:      false,
		},
		{
			name: "comparator filter EqualTo (data: string, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"5.0"},
						Comparator: "=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "5.0"}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "comparator filter empty (data: string, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeNumber,
						Value: []string{"10.0"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "10.0"}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "number filter (data: number, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeNumber,
						Value: []string{"1.0"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": 1.0}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "comparator filter GreaterThan return true (data: number, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"1.0"},
						Comparator: ">",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": 2.0}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "comparator filter LessThanOrEqualTo return false (data: number, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"1.0"},
						Comparator: "<=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": 2.0}`),
				}},
			expectedResult: false,
			expectErr:      false,
		},
		{
			name: "comparator filter NotEqualTo (data: number, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"1.0"},
						Comparator: "!=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": 1.0}`),
				}},
			expectedResult: false,
			expectErr:      false,
		},
		{
			name: "comparator filter EqualTo (data: number, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"5.0"},
						Comparator: "=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": 5.0}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "comparator filter empty (data: number, filter: number), JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeNumber,
						Value: []string{"10.0"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": 10.0}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "multiple filters return false, nested JSON data, EMPTY operator",
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
						Value: []string{"2.14"},
					},
					{
						Path:  "k1.k2",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"hello,world", "hello there"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": true, "k1": {"k": 3.14, "k2": "hello, world"}}`),
				}},
			expectedResult: false,
			expectErr:      false,
		},
		{
			name: "multiple filters return false, nested JSON data, AND operator",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeBool,
						Value: []string{"true"},
					},
					{
						Path:  "k1.k",
						Type:  v1alpha1.JSONTypeNumber,
						Value: []string{"3.14"},
					},
					{
						Path:  "k1.k2",
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"hello,world", "hello, world", "hello there"},
					},
				},
				operator: v1alpha1.AndLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": true, "k1": {"k": 3.14, "k2": "hello, world"}}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "multiple filters return true, nested JSON data, OR operator",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  "k",
						Type:  v1alpha1.JSONTypeBool,
						Value: []string{"false"},
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
				operator: v1alpha1.OrLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": true, "k1": {"k": 3.14, "k2": "hello, world"}}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "string filter Regex, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       `[k,k1.a.#(k2=="v2").k2]`,
						Type:       v1alpha1.JSONTypeString,
						Value:      []string{"\\bv\\b.*\\bv2\\b"},
						Comparator: "=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v", "k1": {"a": [{"k2": "v2"}]}}`),
				},
			},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "string filter Regex2, JSON data",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:  `[k,k1.a.#(k2=="v2").k2,,k1.a.#(k2=="v3").k2]`,
						Type:  v1alpha1.JSONTypeString,
						Value: []string{"(\\bz\\b.*\\bv2\\b)|(\\bv\\b.*(\\bv2\\b.*\\bv3\\b))"},
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "v", "k1": {"a": [{"k2": "v2"}, {"k2": "v3"}]}}`),
				},
			},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "string filter base64, uppercase template",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:     "k",
						Type:     v1alpha1.JSONTypeString,
						Value:    []string{"HELLO WORLD"},
						Template: `{{ b64dec .Input | upper }}`,
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "aGVsbG8gd29ybGQ="}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "string filter base64 template",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"3.13"},
						Comparator: ">",
						Template:   `{{ b64dec .Input }}`,
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "My4xNA=="}`), // 3.14
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "string filter base64 template, comparator not equal",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeString,
						Value:      []string{"hello world"},
						Template:   `{{ b64dec .Input }}`,
						Comparator: "!=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "aGVsbG8gd29ybGQ"}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "string filter base64 template, regex",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:     "k",
						Type:     v1alpha1.JSONTypeString,
						Value:    []string{"world$"},
						Template: `{{ b64dec .Input }}`,
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "aGVsbG8gd29ybGQ="}`),
				}},
			expectedResult: true,
			expectErr:      false,
		},
		{
			name: "string filter NotEqualTo with multiple values - value IN array (should FAIL with fix)",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeString,
						Value:      []string{"id1", "id2", "id3"},
						Comparator: "!=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "id1"}`),
				},
			},
			expectedResult: false, // Should FAIL - value IS in array
			expectErr:      false,
		},
		{
			name: "string filter NotEqualTo with multiple values - value NOT IN array (should PASS)",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeString,
						Value:      []string{"id1", "id2", "id3"},
						Comparator: "!=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": "id_other"}`),
				},
			},
			expectedResult: true, // Should PASS - value NOT in array
			expectErr:      false,
		},
		{
			name: "number filter NotEqualTo with multiple values - value IN array (should FAIL with fix)",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"1.0", "2.0", "3.0"},
						Comparator: "!=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": 2.0}`),
				},
			},
			expectedResult: false, // Should FAIL - value IS in array
			expectErr:      false,
		},
		{
			name: "number filter NotEqualTo with multiple values - value NOT IN array (should PASS)",
			args: args{
				data: []v1alpha1.DataFilter{
					{
						Path:       "k",
						Type:       v1alpha1.JSONTypeNumber,
						Value:      []string{"1.0", "2.0", "3.0"},
						Comparator: "!=",
					},
				},
				operator: v1alpha1.EmptyLogicalOperator,
				event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						DataContentType: "application/json",
					},
					Data: []byte(`{"k": 5.0}`),
				},
			},
			expectedResult: true, // Should PASS - value NOT in array
			expectErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := filterData(test.args.data, test.args.operator, test.args.event)
			if (err != nil) != test.expectErr {
				t.Errorf("filterData() error = %v, expectErr %v", err, test.expectErr)
				return
			}
			if got != test.expectedResult {
				t.Errorf("filterData() = %v, expectedResult %v", got, test.expectedResult)
			}
		})
	}
}
