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
	"io/ioutil"
	"strings"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateSensor(t *testing.T) {
	dir := "../../examples/sensors"
	files, err := ioutil.ReadDir(dir)
	assert.Nil(t, err)
	for _, file := range files {
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
		assert.Nil(t, err)
		var sensor *v1alpha1.Sensor
		err = yaml.Unmarshal(content, &sensor)
		assert.Nil(t, err)
		err = ValidateSensor(sensor)
		assert.Nil(t, err)
	}
}

func TestValidDepencies(t *testing.T) {
	t.Run("test duplicate deps", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			Name:            "fake-dep2",
			EventSourceName: "fake-source",
			EventName:       "fake-one",
		})
		err := ValidateSensor(sObj)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "more than once"))
	})

	t.Run("test empty event source name", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			Name:      "fake-dep2",
			EventName: "fake-one",
		})
		err := ValidateSensor(sObj)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "must define the EventSourceName"))
	})

	t.Run("test empty event name", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			Name:            "fake-dep2",
			EventSourceName: "fake-source",
		})
		err := ValidateSensor(sObj)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "must define the EventName"))
	})

	t.Run("test empty event name", func(t *testing.T) {
		sObj := sensorObj.DeepCopy()
		sObj.Spec.Dependencies = append(sObj.Spec.Dependencies, v1alpha1.EventDependency{
			EventSourceName: "fake-source2",
			EventName:       "fake-one2",
		})
		err := ValidateSensor(sObj)
		assert.NotNil(t, err)
		assert.Equal(t, true, strings.Contains(err.Error(), "must define a name"))
	})
}

func TestValidTriggers(t *testing.T) {
	t.Run("duplicate trigger names", func(t *testing.T) {
		triggers := []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						GroupVersionResource: metav1.GroupVersionResource{
							Group:    "k8s.io",
							Version:  "",
							Resource: "pods",
						},
						Operation: "create",
						Source:    &v1alpha1.ArtifactLocation{},
					},
				},
			},
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8STrigger{
						GroupVersionResource: metav1.GroupVersionResource{
							Group:    "k8s.io",
							Version:  "",
							Resource: "pods",
						},
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
}
