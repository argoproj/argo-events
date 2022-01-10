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

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestFilter(t *testing.T) {
	t.Run("test empty", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := Filter(event, filter, filtersLogicalOperator)

		assert.NoError(t, err)
		assert.True(t, pass)
	})

	t.Run("test event passing", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, pass)
	})

	t.Run("test event not passing", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			Data: []v1alpha1.DataFilter{
				{
					Path:  "z",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, pass)
	})

	t.Run("test error", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: "09:09:0",
				Stop:  "19:19:19",
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, pass)
	})

	t.Run("test 'empty' filtersLogicalOperator", func(t *testing.T) {
		// ctx filter: true
		// data filter: false
		filter := &v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"z"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.EmptyLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, pass)
	})

	t.Run("test 'empty' filtersLogicalOperator with error", func(t *testing.T) {
		// ctx filter: true
		// data filter: error (false)
		filter := &v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "z",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.EmptyLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, pass)
	})

	t.Run("test 'and' filtersLogicalOperator", func(t *testing.T) {
		// ctx filter: false
		// data filter: true
		filter := &v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-fake",
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, pass)
	})

	t.Run("test 'and' filtersLogicalOperator with error", func(t *testing.T) {
		// ctx filter: true
		// data filter: error (false)
		filter := &v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "z",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, pass)
	})

	t.Run("test 'or' filtersLogicalOperator", func(t *testing.T) {
		// ctx filter: true
		// data filter: false
		filter := &v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "z",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.OrLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.True(t, pass)
	})

	t.Run("test 'or' filtersLogicalOperator with error", func(t *testing.T) {
		// ctx filter: true
		// data filter: error (false)
		filter := &v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "z",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.OrLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			Data: []byte(`{"k": "v"}`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.True(t, pass)
	})

	t.Run("test advanced logic: (A && B) && (C && D)", func(t *testing.T) {
		// data filter: A && B == true
		// expr filter: C && D == true
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.EmptyLogicalOperator, // default AND
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "a",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"10"},
				},
			},
			ExprLogicalOperator: v1alpha1.EmptyLogicalOperator, // default AND
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `d == "hello world"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.d",
							Name: "d",
						},
					},
				},
				{
					// D
					Expr: `e == false`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.EmptyLogicalOperator // default AND
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, pass)
	})

	t.Run("test advanced logic: (A && B) && (C && D) with error", func(t *testing.T) {
		// data filter: A && B == error (false)
		// expr filter: C && D == true
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.EmptyLogicalOperator, // default AND
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "z",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"10"},
				},
			},
			ExprLogicalOperator: v1alpha1.EmptyLogicalOperator, // default AND
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `d == "hello world"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.d",
							Name: "d",
						},
					},
				},
				{
					// D
					Expr: `e == false"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.EmptyLogicalOperator // default AND
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, pass)
	})

	t.Run("test advanced logic: (A && B) || (C && D)", func(t *testing.T) {
		// data filter: A && B == true
		// expr filter: C && D == false
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.AndLogicalOperator,
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "a",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"10"},
				},
			},
			ExprLogicalOperator: v1alpha1.AndLogicalOperator,
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `d == "hello world"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.d",
							Name: "d",
						},
					},
				},
				{
					// D
					Expr: `e == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.OrLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, pass)
	})

	t.Run("test advanced logic: (A || B) && (C || D)", func(t *testing.T) {
		// data filter: A || B == true
		// expr filter: C || D == false
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.OrLogicalOperator,
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "a",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"11"},
				},
			},
			ExprLogicalOperator: v1alpha1.OrLogicalOperator,
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `d == "hello world"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.d",
							Name: "d",
						},
					},
				},
				{
					// D
					Expr: `e == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, pass)
	})

	t.Run("test advanced logic: (A || B) || (C || D)", func(t *testing.T) {
		// data filter: A || B == false
		// expr filter: C || D == true
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.OrLogicalOperator,
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "a",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"y"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"11"},
				},
			},
			ExprLogicalOperator: v1alpha1.OrLogicalOperator,
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `d == "hello world"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.d",
							Name: "d",
						},
					},
				},
				{
					// D
					Expr: `e == false`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.OrLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, pass)
	})

	t.Run("test advanced logic: (A && B) || (C || D)", func(t *testing.T) {
		// data filter: A && B == true
		// expr filter: C || D == false
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.AndLogicalOperator,
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "a",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"10"},
				},
			},
			ExprLogicalOperator: v1alpha1.OrLogicalOperator,
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `d == "hello everybody"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.d",
							Name: "d",
						},
					},
				},
				{
					// D
					Expr: `e == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.OrLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, pass)
	})

	t.Run("test advanced logic: (A || B) || (C && D)", func(t *testing.T) {
		// data filter: A || B == true
		// expr filter: C && D == false
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.OrLogicalOperator,
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "a",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"11"},
				},
			},
			ExprLogicalOperator: v1alpha1.AndLogicalOperator,
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `d == "hello everybody"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.d",
							Name: "d",
						},
					},
				},
				{
					// D
					Expr: `e == false`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.OrLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, pass)
	})

	t.Run("test advanced logic: (A && B) && (C || D)", func(t *testing.T) {
		// data filter: A && B == false
		// expr filter: C || D == false
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.AndLogicalOperator,
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "a",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"11"},
				},
			},
			ExprLogicalOperator: v1alpha1.OrLogicalOperator,
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `d == "hello everybody"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.d",
							Name: "d",
						},
					},
				},
				{
					// D
					Expr: `e == true`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, pass)
	})

	t.Run("test advanced logic: (A || B) && (C && D)", func(t *testing.T) {
		// data filter: A || B == true
		// expr filter: C && D == error (false)
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: v1alpha1.OrLogicalOperator,
			Data: []v1alpha1.DataFilter{
				{
					// A
					Path:  "a",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
				{
					// B
					Path:  "b",
					Type:  v1alpha1.JSONTypeNumber,
					Value: []string{"10"},
				},
			},
			ExprLogicalOperator: v1alpha1.AndLogicalOperator,
			Exprs: []v1alpha1.ExprFilter{
				{
					// C
					Expr: `f == "hello world"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.f",
							Name: "f",
						},
					},
				},
				{
					// D
					Expr: `e == false`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "c.e",
							Name: "e",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator
		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
			},
			Data: []byte(`{ "a": "x", "b": 10, "c": { "d": "hello world", "e": false } }`),
		}

		pass, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, pass)
	})

}
