package dependencies

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestFilterEvent_All(t *testing.T) {
	t.Run("test all valid", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: testStartTime,
				Stop:  testStopTime,
			},
			Context: &v1alpha1.EventContext{
				Type:   testEventType,
				Source: testSource,
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: testExprKNotX,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "k",
							Name: "k",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("test time not valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: testStartTime,
				Stop:  "10:10:10",
			},
			Context: &v1alpha1.EventContext{
				Type:   testEventType,
				Source: testSource,
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: testExprKNotX,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "k",
							Name: "k",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("test ctx not valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: testStartTime,
				Stop:  testStopTime,
			},
			Context: &v1alpha1.EventContext{
				Type:   testEventType,
				Source: testSource,
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: testExprKNotX,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "k",
							Name: "k",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testWebhookFake,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("test data not valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: testStartTime,
				Stop:  testStopTime,
			},
			Context: &v1alpha1.EventContext{
				Type:   testEventType,
				Source: testSource,
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: `k == "x"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "k",
							Name: "k",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "x"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("test expr not valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: testStartTime,
				Stop:  testStopTime,
			},
			Context: &v1alpha1.EventContext{
				Type:   testEventType,
				Source: testSource,
			},
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"v"},
				},
			},
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: `k != "v"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "k",
							Name: "k",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, valid)
	})
}

func TestFilterEvent_Expr(t *testing.T) {
	t.Run("test valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: testExprKNotX,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "k",
							Name: "k",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("test not valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: `k != "v"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "k",
							Name: "k",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("test error", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: `k !== "x"`,
					Fields: []v1alpha1.PayloadField{
						{
							Path: "k",
							Name: "k",
						},
					},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, valid)
	})
}

func TestFilterEvent_Data(t *testing.T) {
	t.Run("test valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
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
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("test not valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Data: []v1alpha1.DataFilter{
				{
					Path:  "k",
					Type:  v1alpha1.JSONTypeString,
					Value: []string{"x"},
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("test error", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Data: []v1alpha1.DataFilter{
				{
					Path: "k",
					Type: v1alpha1.JSONTypeString,
				},
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, valid)
	})
}

func TestFilterEvent_Context(t *testing.T) {
	t.Run("test valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:   testEventType,
				Source: testSource,
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("test not valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:   testEventType,
				Source: testWebhookFake,
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, valid)
	})
}

func TestFilterEvent_Time(t *testing.T) {
	t.Run("test valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: testStartTime,
				Stop:  testStopTime,
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("test not valid", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: testStartTime,
				Stop:  "10:10:10",
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("test error", func(t *testing.T) {
		filter := v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: testStartTime,
				Stop:  "10:10",
			},
		}
		filtersLogicalOperator := v1alpha1.AndLogicalOperator

		now := time.Now().UTC()
		event := &v1alpha1.Event{
			Context: &v1alpha1.EventContext{
				Type:        testEventType,
				SpecVersion: testSpecVersion,
				Source:      testSource,
				ID:          "1",
				Time: metav1.Time{
					Time: time.Date(now.Year(), now.Month(), now.Day(), 16, 36, 34, 0, time.UTC),
				},
				DataContentType: testContentType,
				Subject:         testSubject,
			},
			Data: []byte(`{"k": "v"}`),
		}

		valid, err := filterEvent(&filter, filtersLogicalOperator, event)

		assert.Error(t, err)
		assert.False(t, valid)
	})
}
