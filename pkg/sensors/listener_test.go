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
	"fmt"
	"testing"

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

func TestShouldExecuteByWeight(t *testing.T) {
	t.Run("weight not set - should always execute", func(t *testing.T) {
		trigger := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-1"},
			Weight:   0,
		}
		allTriggers := []v1alpha1.Trigger{trigger}

		shouldExecute, weightEnabled := shouldExecuteByWeight(trigger, allTriggers, 0)
		assert.True(t, shouldExecute)
		assert.False(t, weightEnabled)
	})

	t.Run("single trigger with weight 100 - should always execute", func(t *testing.T) {
		trigger := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-1"},
			Weight:   100,
		}
		allTriggers := []v1alpha1.Trigger{trigger}

		for i := uint64(0); i < 10; i++ {
			shouldExecute, weightEnabled := shouldExecuteByWeight(trigger, allTriggers, i)
			assert.True(t, shouldExecute, "trigger should execute for hash %d", i)
			assert.True(t, weightEnabled)
		}
	})

	t.Run("two triggers with equal weights - correct distribution", func(t *testing.T) {
		trigger1 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-1"},
			Weight:   50,
		}
		trigger2 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-2"},
			Weight:   50,
		}
		allTriggers := []v1alpha1.Trigger{trigger1, trigger2}

		// Hash 0-49 should go to trigger-1, 50-99 should go to trigger-2
		for i := uint64(0); i < 50; i++ {
			shouldExecute, weightEnabled := shouldExecuteByWeight(trigger1, allTriggers, i)
			assert.True(t, shouldExecute, "trigger-1 should execute for hash %d", i)
			assert.True(t, weightEnabled)
		}
		for i := uint64(50); i < 100; i++ {
			shouldExecute, weightEnabled := shouldExecuteByWeight(trigger2, allTriggers, i)
			assert.True(t, shouldExecute, "trigger-2 should execute for hash %d", i)
			assert.True(t, weightEnabled)
		}
	})

	t.Run("two triggers with 30/70 weights", func(t *testing.T) {
		trigger1 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-1"},
			Weight:   30,
		}
		trigger2 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-2"},
			Weight:   70,
		}
		allTriggers := []v1alpha1.Trigger{trigger1, trigger2}

		trigger1Count := 0
		trigger2Count := 0
		for i := uint64(0); i < 100; i++ {
			shouldExecute1, _ := shouldExecuteByWeight(trigger1, allTriggers, i)
			shouldExecute2, _ := shouldExecuteByWeight(trigger2, allTriggers, i)

			if shouldExecute1 {
				trigger1Count++
			}
			if shouldExecute2 {
				trigger2Count++
			}
		}

		assert.Equal(t, 30, trigger1Count, "trigger-1 should execute 30 times")
		assert.Equal(t, 70, trigger2Count, "trigger-2 should execute 70 times")
	})

	t.Run("mixed weighted and non-weighted triggers", func(t *testing.T) {
		trigger1 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-1"},
			Weight:   30,
		}
		trigger2 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-2"},
			Weight:   70,
		}
		trigger3 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-3"},
			Weight:   0, // Non-weighted, should always execute
		}
		allTriggers := []v1alpha1.Trigger{trigger1, trigger2, trigger3}

		// Non-weighted trigger should always execute
		for i := uint64(0); i < 10; i++ {
			shouldExecute, weightEnabled := shouldExecuteByWeight(trigger3, allTriggers, i)
			assert.True(t, shouldExecute, "non-weighted trigger should always execute")
			assert.False(t, weightEnabled)
		}
	})

	t.Run("mutually exclusive execution for weighted triggers", func(t *testing.T) {
		trigger1 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-1"},
			Weight:   30,
		}
		trigger2 := v1alpha1.Trigger{
			Template: &v1alpha1.TriggerTemplate{Name: "trigger-2"},
			Weight:   70,
		}
		allTriggers := []v1alpha1.Trigger{trigger1, trigger2}

		// For any given hash, exactly one weighted trigger should execute
		for i := uint64(0); i < 100; i++ {
			shouldExecute1, _ := shouldExecuteByWeight(trigger1, allTriggers, i)
			shouldExecute2, _ := shouldExecuteByWeight(trigger2, allTriggers, i)

			// XOR: exactly one should be true
			assert.True(t, shouldExecute1 != shouldExecute2,
				"exactly one trigger should execute for hash %d, got trigger1=%v, trigger2=%v",
				i, shouldExecute1, shouldExecute2)
		}
	})
}

func TestHashEventIDs(t *testing.T) {
	t.Run("same event ID produces same hash", func(t *testing.T) {
		event1 := cloudevents.NewEvent()
		event1.SetID("test-event-123")

		events := map[string]cloudevents.Event{"dep1": event1}

		hash1 := hashEventIDs(events)
		hash2 := hashEventIDs(events)

		assert.Equal(t, hash1, hash2, "same events should produce same hash")
	})

	t.Run("different event IDs produce different hashes", func(t *testing.T) {
		event1 := cloudevents.NewEvent()
		event1.SetID("test-event-123")

		event2 := cloudevents.NewEvent()
		event2.SetID("test-event-456")

		events1 := map[string]cloudevents.Event{"dep1": event1}
		events2 := map[string]cloudevents.Event{"dep1": event2}

		hash1 := hashEventIDs(events1)
		hash2 := hashEventIDs(events2)

		assert.NotEqual(t, hash1, hash2, "different events should produce different hashes")
	})

	t.Run("order of events in map does not affect hash", func(t *testing.T) {
		event1 := cloudevents.NewEvent()
		event1.SetID("event-a")
		event2 := cloudevents.NewEvent()
		event2.SetID("event-b")

		// Maps are unordered, but our hash should be deterministic
		events := map[string]cloudevents.Event{
			"dep1": event1,
			"dep2": event2,
		}

		hash1 := hashEventIDs(events)
		hash2 := hashEventIDs(events)

		assert.Equal(t, hash1, hash2, "hash should be deterministic regardless of map iteration order")
	})

	t.Run("good distribution of hashes", func(t *testing.T) {
		// Test that hash function provides reasonable distribution
		buckets := make([]int, 100)

		for i := 0; i < 1000; i++ {
			event := cloudevents.NewEvent()
			event.SetID(fmt.Sprintf("event-%d", i))
			events := map[string]cloudevents.Event{"dep1": event}

			hash := hashEventIDs(events)
			buckets[hash%100]++
		}

		// Check that no bucket is completely empty (rough distribution test)
		emptyBuckets := 0
		for _, count := range buckets {
			if count == 0 {
				emptyBuckets++
			}
		}

		// With 1000 events into 100 buckets, we expect most buckets to have some events
		assert.Less(t, emptyBuckets, 20, "hash should distribute reasonably well")
	})
}
