/*
Copyright 2018 The Argoproj Authors.

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
	"os"
	"strings"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSensor(t *testing.T) {
	dir := "../../../examples/sensors"
	dirEntries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		t.Run(
			fmt.Sprintf("test example load: %s/%s", dir, entry.Name()),
			func(t *testing.T) {
				content, err := os.ReadFile(fmt.Sprintf("%s/%s", dir, entry.Name()))
				assert.NoError(t, err)

				var sensor *v1alpha1.Sensor
				eventBus := &v1alpha1.EventBus{Spec: v1alpha1.EventBusSpec{JetStream: &v1alpha1.JetStreamBus{}}}
				err = yaml.Unmarshal(content, &sensor)
				assert.NoError(t, err)

				err = ValidateSensor(sensor, eventBus)
				assert.NoError(t, err)
			})
	}
}

func TestValidDependencies(t *testing.T) {
	jetstreamBus := &v1alpha1.EventBus{Spec: v1alpha1.EventBusSpec{JetStream: &v1alpha1.JetStreamBus{}}}
	stanBus := &v1alpha1.EventBus{Spec: v1alpha1.EventBusSpec{NATS: &v1alpha1.NATSBus{}}}

	t.Run("test duplicate deps fail for STAN", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			Name:            "fake-dep2",
			EventSourceName: "fake-source",
			EventName:       "fake-one",
		})
		err := ValidateSensor(sObj, stanBus)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "is referenced for more than one dependency"))
	})

	t.Run("test duplicate deps are fine for Jetstream", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			Name:            "fake-dep2",
			EventSourceName: "fake-source",
			EventName:       "fake-one",
		})
		err := ValidateSensor(sObj, jetstreamBus)
		assert.Nil(t, err)
	})

	t.Run("test empty event source name", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			Name:      "fake-dep2",
			EventName: "fake-one",
		})
		err := ValidateSensor(sObj, jetstreamBus)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "must define the EventSourceName"))
	})

	t.Run("test empty event name", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			Name:            "fake-dep2",
			EventSourceName: "fake-source",
		})
		err := ValidateSensor(sObj, jetstreamBus)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "must define the EventName"))
	})

	t.Run("test empty event name", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			EventSourceName: "fake-source2",
			EventName:       "fake-one2",
		})
		err := ValidateSensor(sObj, jetstreamBus)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "must define a name"))
	})
}

func TestValidateLogicalOperator(t *testing.T) {
	t.Run("test valid", func(t *testing.T) {
		logOp := v1alpha1.OrLogicalOperator

		err := validateLogicalOperator(logOp)

		assert.NoError(t, err)
	})

	t.Run("test not valid", func(t *testing.T) {
		logOp := v1alpha1.LogicalOperator("fake")

		err := validateLogicalOperator(logOp)

		assert.Error(t, err)
	})
}

func TestValidateComparator(t *testing.T) {
	t.Run("test valid", func(t *testing.T) {
		comp := v1alpha1.NotEqualTo

		err := validateComparator(comp)

		assert.NoError(t, err)
	})

	t.Run("test not valid", func(t *testing.T) {
		comp := v1alpha1.Comparator("fake")

		err := validateComparator(comp)

		assert.Error(t, err)
	})
}

func TestValidateEventFilter(t *testing.T) {
	t.Run("test empty", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{}

		err := validateEventFilter(filter)

		assert.NoError(t, err)
	})

	t.Run("test valid, all", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			ExprLogicalOperator: v1alpha1.OrLogicalOperator,
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: "fake-expr",
					Fields: []v1alpha1.PayloadField{
						{
							Path: "fake-path",
							Name: "fake-name",
						},
					},
				},
			},
			DataLogicalOperator: v1alpha1.OrLogicalOperator,
			Data: []v1alpha1.DataFilter{
				{
					Path: "fake-path",
					Type: "fake-type",
					Value: []string{
						"fake-value",
					},
				},
			},
			Context: &v1alpha1.EventContext{
				Type:            "type",
				Subject:         "subject",
				Source:          "source",
				DataContentType: "fake-content-type",
			},
			Time: &v1alpha1.TimeFilter{
				Start: "00:00:00",
				Stop:  "06:00:00",
			},
		}

		err := validateEventFilter(filter)

		assert.NoError(t, err)
	})

	t.Run("test valid, expr only", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			Exprs: []v1alpha1.ExprFilter{
				{
					Expr: "fake-expr",
					Fields: []v1alpha1.PayloadField{
						{
							Path: "fake-path",
							Name: "fake-name",
						},
					},
				},
			},
		}

		err := validateEventFilter(filter)

		assert.NoError(t, err)
	})

	t.Run("test valid, data only", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			Data: []v1alpha1.DataFilter{
				{
					Path: "fake-path",
					Type: "fake-type",
					Value: []string{
						"fake-value",
					},
				},
			},
		}

		err := validateEventFilter(filter)

		assert.NoError(t, err)
	})

	t.Run("test valid, ctx only", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			Context: &v1alpha1.EventContext{
				Type:            "type",
				Subject:         "subject",
				Source:          "source",
				DataContentType: "fake-content-type",
			},
		}

		err := validateEventFilter(filter)

		assert.NoError(t, err)
	})

	t.Run("test valid, time only", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			Time: &v1alpha1.TimeFilter{
				Start: "00:00:00",
				Stop:  "06:00:00",
			},
		}

		err := validateEventFilter(filter)

		assert.NoError(t, err)
	})

	t.Run("test not valid, wrong logical operator", func(t *testing.T) {
		filter := &v1alpha1.EventDependencyFilter{
			DataLogicalOperator: "fake",
		}

		err := validateEventFilter(filter)

		assert.Error(t, err)
	})
}

func TestValidateEventExprFilter(t *testing.T) {
	t.Run("test valid", func(t *testing.T) {
		exprFilter := &v1alpha1.ExprFilter{
			Expr: "fake-expr",
			Fields: []v1alpha1.PayloadField{
				{
					Path: "fake-path",
					Name: "fake-name",
				},
			},
		}

		err := validateEventExprFilter(exprFilter)

		assert.NoError(t, err)
	})

	t.Run("test not valid, no expr", func(t *testing.T) {
		exprFilter := &v1alpha1.ExprFilter{
			Fields: []v1alpha1.PayloadField{
				{
					Path: "fake-path",
					Name: "fake-name",
				},
			},
		}

		err := validateEventExprFilter(exprFilter)

		assert.Error(t, err)
	})

	t.Run("test not valid, no field name", func(t *testing.T) {
		exprFilter := &v1alpha1.ExprFilter{
			Expr: "fake-expr",
			Fields: []v1alpha1.PayloadField{
				{
					Path: "fake-path",
				},
			},
		}

		err := validateEventExprFilter(exprFilter)

		assert.Error(t, err)
	})
}

func TestValidateEventDataFilter(t *testing.T) {
	t.Run("test valid", func(t *testing.T) {
		dataFilter := &v1alpha1.DataFilter{
			Path:  "body.value",
			Type:  "number",
			Value: []string{"50.0"},
		}

		err := validateEventDataFilter(dataFilter)

		assert.NoError(t, err)
	})

	t.Run("test not valid, no path", func(t *testing.T) {
		dataFilter := &v1alpha1.DataFilter{
			Type:  "number",
			Value: []string{"50.0"},
		}

		err := validateEventDataFilter(dataFilter)

		assert.Error(t, err)
	})

	t.Run("test not valid, empty value", func(t *testing.T) {
		dataFilter := &v1alpha1.DataFilter{
			Path:  "body.value",
			Type:  "string",
			Value: []string{""},
		}

		err := validateEventDataFilter(dataFilter)

		assert.Error(t, err)
	})

	t.Run("test not valid, wrong comparator", func(t *testing.T) {
		dataFilter := &v1alpha1.DataFilter{
			Comparator: "fake",
			Path:       "body.value",
			Type:       "string",
			Value:      []string{""},
		}

		err := validateEventDataFilter(dataFilter)

		assert.Error(t, err)
	})
}

func TestValidateEventCtxFilter(t *testing.T) {
	t.Run("test all fields", func(t *testing.T) {
		ctxFilter := &v1alpha1.EventContext{
			Type:            "fake-type",
			Subject:         "fake-subject",
			Source:          "fake-source",
			DataContentType: "fake-content-type",
		}

		err := validateEventCtxFilter(ctxFilter)

		assert.NoError(t, err)
	})

	t.Run("test single field", func(t *testing.T) {
		ctxFilter := &v1alpha1.EventContext{
			Type: "fake-type",
		}

		err := validateEventCtxFilter(ctxFilter)

		assert.NoError(t, err)
	})

	t.Run("test no fields", func(t *testing.T) {
		ctxFilter := &v1alpha1.EventContext{}

		err := validateEventCtxFilter(ctxFilter)

		assert.Error(t, err)
	})
}

func TestValidateEventTimeFilter(t *testing.T) {
	t.Run("test start < stop", func(t *testing.T) {
		timeFilter := &v1alpha1.TimeFilter{
			Start: "00:00:00",
			Stop:  "06:00:00",
		}

		err := validateEventTimeFilter(timeFilter)

		assert.NoError(t, err)
	})

	t.Run("test stop < start", func(t *testing.T) {
		timeFilter := &v1alpha1.TimeFilter{
			Start: "06:00:00",
			Stop:  "00:00:00",
		}

		err := validateEventTimeFilter(timeFilter)

		assert.NoError(t, err)
	})

	t.Run("test start = stop", func(t *testing.T) {
		timeFilter := &v1alpha1.TimeFilter{
			Start: "00:00:00",
			Stop:  "00:00:00",
		}

		err := validateEventTimeFilter(timeFilter)

		assert.Error(t, err)
	})

	t.Run("test with valid timezone", func(t *testing.T) {
		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "America/New_York",
		}

		err := validateEventTimeFilter(timeFilter)

		assert.NoError(t, err)
	})

	t.Run("test with invalid timezone", func(t *testing.T) {
		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "Invalid/Timezone",
		}

		err := validateEventTimeFilter(timeFilter)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid timezone")
	})

	t.Run("test with empty timezone (defaults to UTC)", func(t *testing.T) {
		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "",
		}

		err := validateEventTimeFilter(timeFilter)

		assert.NoError(t, err)
	})
}

func TestValidTriggers(t *testing.T) {
	t.Run("duplicate trigger names", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
			},
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
			},
		}
		err := validateTriggers(triggers)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "duplicate trigger name:"))
	})

	t.Run("empty trigger template", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: nil,
			},
		}
		err := validateTriggers(triggers)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "trigger template can't be nil"))
	})

	t.Run("vanilla dlqTrigger", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
				AtLeastOnce: true,
				DlqTrigger: &v1alpha1.Trigger{
					AtLeastOnce: true,
					Template: &v1alpha1.TriggerTemplate{
						Name: "dlq-fake-trigger",
						K8s: &v1alpha1.StandardK8STrigger{
							Operation: "create",
							Source:    &v1alpha1.ArtifactLocation{},
						},
					},
				},
			},
		}
		err := validateTriggers(triggers)
		assert.Nil(t, err)
	})

	t.Run("!dlqTrigger.atLeastOnce", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
				AtLeastOnce: true,
				DlqTrigger: &v1alpha1.Trigger{
					Template: &v1alpha1.TriggerTemplate{
						Name: "dlq-fake-trigger",
						K8s: &v1alpha1.StandardK8STrigger{
							Operation: "create",
							Source:    &v1alpha1.ArtifactLocation{},
						},
					},
				},
			},
		}
		err := validateTriggers(triggers)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "atLeastOnce must be set to true within the dlqTrigger"))
	})

	t.Run("dlqTrigger !.atLeastOnce", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
				DlqTrigger: &v1alpha1.Trigger{
					Template: &v1alpha1.TriggerTemplate{
						Name: "dlq-fake-trigger",
						K8s: &v1alpha1.StandardK8STrigger{
							Operation: "create",
							Source:    &v1alpha1.ArtifactLocation{},
						},
					},
				},
			},
		}
		err := validateTriggers(triggers)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "to use dlqTrigger, trigger.atLeastOnce must be set to true"))
	})

	t.Run("invalid conditions reset - cron", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name:       "fake-trigger",
					Conditions: "A && B",
					ConditionsReset: []v1alpha1.ConditionsResetCriteria{
						{
							ByTime: &v1alpha1.ConditionsResetByTime{
								Cron: "a * * * *",
							},
						},
					},
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
			},
		}
		err := validateTriggers(triggers)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "invalid cron expression"))
	})

	t.Run("invalid conditions reset - timezone", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name:       "fake-trigger",
					Conditions: "A && B",
					ConditionsReset: []v1alpha1.ConditionsResetCriteria{
						{
							ByTime: &v1alpha1.ConditionsResetByTime{
								Cron:     "* * * * *",
								Timezone: "fake",
							},
						},
					},
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
			},
		}
		err := validateTriggers(triggers)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "invalid timezone"))
	})

	t.Run("valid trigger weight", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger-1",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
				Weight: 30,
			},
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger-2",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
				Weight: 70,
			},
		}
		err := validateTriggers(triggers)
		assert.Nil(t, err)
	})

	t.Run("negative trigger weight", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
				Weight: -10,
			},
		}
		err := validateTriggers(triggers)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "trigger weight must be non-negative"))
	})

	t.Run("trigger weight exceeds 100", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
				Weight: 150,
			},
		}
		err := validateTriggers(triggers)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "trigger weight must not exceed 100"))
	})

	t.Run("zero weight is valid (weight-based routing disabled)", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
				Weight: 0,
			},
		}
		err := validateTriggers(triggers)
		assert.Nil(t, err)
	})
}
