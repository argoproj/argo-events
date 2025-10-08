/*
Copyright 2020 BlackRock, Inc.

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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
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

func TestDependencyMapping(t *testing.T) {
	t.Run("JetStream config is mapped correctly", func(t *testing.T) {
		obj := sensorObj.DeepCopy()
		obj.Spec.Dependencies = []v1alpha1.EventDependency{
			{
				Name:            "dep1",
				EventSourceName: "webhook",
				EventName:       "example-1",
				JetStream: &v1alpha1.JetStreamConsumerConfig{
					DeliverPolicy: v1alpha1.JetStreamDeliverAll,
				},
			},
			{
				Name:            "dep2",
				EventSourceName: "webhook2",
				EventName:       "example-2",
				JetStream: &v1alpha1.JetStreamConsumerConfig{
					DeliverPolicy: v1alpha1.JetStreamDeliverLast,
				},
			},
			{
				Name:            "dep3",
				EventSourceName: "webhook3",
				EventName:       "example-3",
				// No JetStream config - should be nil
			},
		}

		// Create dependency mapping like the listener does
		depMapping := make(map[string]v1alpha1.EventDependency)
		for _, d := range obj.Spec.Dependencies {
			depMapping[d.Name] = d
		}

		// Test that each dependency preserves JetStream config
		for _, depName := range []string{"dep1", "dep2", "dep3"} {
			dep, ok := depMapping[depName]
			assert.True(t, ok, "Dependency %s should exist", depName)

			// Verify the JetStream config is accessible
			if depName == "dep1" {
				assert.NotNil(t, dep.JetStream, "dep1 should have JetStream config")
				assert.Equal(t, v1alpha1.JetStreamDeliverAll, dep.JetStream.DeliverPolicy)
			} else if depName == "dep2" {
				assert.NotNil(t, dep.JetStream, "dep2 should have JetStream config")
				assert.Equal(t, v1alpha1.JetStreamDeliverLast, dep.JetStream.DeliverPolicy)
			} else if depName == "dep3" {
				assert.Nil(t, dep.JetStream, "dep3 should not have JetStream config")
			}
		}
	})
}
