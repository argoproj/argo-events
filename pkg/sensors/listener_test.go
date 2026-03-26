/*
Copyright 2020 The Argoproj Authors.

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
	"context"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

var (
	fakeTrigger = &v1alpha1.Trigger{
		Template: &v1alpha1.TriggerTemplate{
			Name: "fake-trigger",
			K8s:  &v1alpha1.StandardK8STrigger{},
		},
	}

	sensorObj = &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-sensor",
			Namespace: "fake",
		},
		Spec: v1alpha1.SensorSpec{
			Triggers: []v1alpha1.Trigger{
				*fakeTrigger,
			},
		},
	}
)

func TestGetDependencyExpression(t *testing.T) {
	t.Run("get simple expression", func(t *testing.T) {
		obj := sensorObj.DeepCopy()
		obj.Spec.Dependencies = []v1alpha1.EventDependency{
			{
				Name:            "dep1",
				EventSourceName: "webhook",
				EventName:       "example-1",
			},
		}
		sensorCtx := &SensorContext{
			sensor: obj,
		}
		expr, err := sensorCtx.getDependencyExpression(context.Background(), *fakeTrigger)
		assert.NoError(t, err)
		assert.Equal(t, "dep1", expr)
	})

	t.Run("get two deps expression", func(t *testing.T) {
		obj := sensorObj.DeepCopy()
		obj.Spec.Dependencies = []v1alpha1.EventDependency{
			{
				Name:            "dep1",
				EventSourceName: "webhook",
				EventName:       "example-1",
			},
			{
				Name:            "dep2",
				EventSourceName: "webhook2",
				EventName:       "example-2",
			},
		}
		sensorCtx := &SensorContext{
			sensor: obj,
		}
		_, err := sensorCtx.getDependencyExpression(context.Background(), *fakeTrigger)
		assert.NoError(t, err)
	})

	t.Run("get complex expression", func(t *testing.T) {
		obj := sensorObj.DeepCopy()
		obj.Spec.Dependencies = []v1alpha1.EventDependency{
			{
				Name:            "dep1",
				EventSourceName: "webhook",
				EventName:       "example-1",
			},
			{
				Name:            "dep1a",
				EventSourceName: "webhook",
				EventName:       "example-1a",
			},
			{
				Name:            "dep2",
				EventSourceName: "webhook2",
				EventName:       "example-2",
			},
		}
		sensorCtx := &SensorContext{
			sensor: obj,
		}
		trig := fakeTrigger.DeepCopy()
		_, err := sensorCtx.getDependencyExpression(context.Background(), *trig)
		assert.NoError(t, err)
	})

	t.Run("get conditions expression", func(t *testing.T) {
		obj := sensorObj.DeepCopy()
		obj.Spec.Dependencies = []v1alpha1.EventDependency{
			{
				Name:            "dep-1",
				EventSourceName: "webhook",
				EventName:       "example-1",
			},
			{
				Name:            "dep_1a",
				EventSourceName: "webhook",
				EventName:       "example-1a",
			},
			{
				Name:            "dep-2",
				EventSourceName: "webhook2",
				EventName:       "example-2",
			},
			{
				Name:            "dep-3",
				EventSourceName: "webhook3",
				EventName:       "example-3",
			},
		}
		sensorCtx := &SensorContext{
			sensor: obj,
		}
		trig := fakeTrigger.DeepCopy()
		trig.Template.Conditions = "dep-1 || dep-1a || dep-3"
		_, err := sensorCtx.getDependencyExpression(context.Background(), *trig)
		assert.NoError(t, err)
	})
}

func TestConvertEvent(t *testing.T) {
	t.Run("converts standard attributes", func(t *testing.T) {
		ce := cloudevents.NewEvent()
		ce.SetID("test-id")
		ce.SetSource("test-source")
		ce.SetType("test-type")
		ce.SetSubject("test-subject")
		ce.SetDataContentType("application/json")
		ts := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
		ce.SetTime(ts)
		_ = ce.SetData(cloudevents.ApplicationJSON, map[string]string{"key": "value"})

		result := convertEvent(ce)

		assert.Equal(t, "test-id", result.Context.ID)
		assert.Equal(t, "test-source", result.Context.Source)
		assert.Equal(t, "test-type", result.Context.Type)
		assert.Equal(t, "test-subject", result.Context.Subject)
		assert.Equal(t, "application/json", result.Context.DataContentType)
		assert.Equal(t, ts, result.Context.Time.Time)
		assert.NotEmpty(t, result.Data)
	})

	t.Run("preserves extensions", func(t *testing.T) {
		ce := cloudevents.NewEvent()
		ce.SetID("test-id")
		ce.SetSource("test-source")
		ce.SetType("test-type")
		ce.SetExtension("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		ce.SetExtension("tracestate", "congo=t61rcWkgMzE")
		ce.SetExtension("customext", "myvalue")

		result := convertEvent(ce)

		assert.NotNil(t, result.Context.Extensions)
		assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", result.Context.Extensions["traceparent"])
		assert.Equal(t, "congo=t61rcWkgMzE", result.Context.Extensions["tracestate"])
		assert.Equal(t, "myvalue", result.Context.Extensions["customext"])
	})

	t.Run("no extensions results in nil map", func(t *testing.T) {
		ce := cloudevents.NewEvent()
		ce.SetID("test-id")
		ce.SetSource("test-source")
		ce.SetType("test-type")

		result := convertEvent(ce)

		assert.Nil(t, result.Context.Extensions)
	})
}
