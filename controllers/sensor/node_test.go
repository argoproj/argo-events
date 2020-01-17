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
	"testing"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	fakesensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSensorState(t *testing.T) {
	fakeSensorClient := fakesensor.NewSimpleClientset()
	logger := common.NewArgoEventsLogger()
	fakeSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sensor",
			Namespace: "test",
		},
	}

	fakeSensor, err := fakeSensorClient.ArgoprojV1alpha1().Sensors(fakeSensor.Namespace).Create(fakeSensor)
	assert.Nil(t, err)

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "initialize a new node",
			testFunc: func(t *testing.T) {
				status := InitializeNode(fakeSensor, "first_node", v1alpha1.NodeTypeEventDependency, logger)
				assert.Equal(t, status.Phase, v1alpha1.NodePhaseNew)
			},
		},
		{
			name: "persist updates to the sensor",
			testFunc: func(t *testing.T) {
				sensor, err := PersistUpdates(fakeSensorClient, fakeSensor, logger)
				assert.Nil(t, err)
				assert.Equal(t, len(sensor.Status.Nodes), 1)
			},
		},
		{
			name: "mark node state to active",
			testFunc: func(t *testing.T) {
				status := MarkNodePhase(fakeSensor, "first_node", v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, &apicommon.Event{
					Data: []byte("test payload"),
				}, logger)
				assert.Equal(t, status.Phase, v1alpha1.NodePhaseActive)
			},
		},
		{
			name: "reapply the update",
			testFunc: func(t *testing.T) {
				err := ReapplyUpdate(fakeSensorClient, fakeSensor)
				assert.Nil(t, err)
			},
		},
		{
			name: "fetch sensor and check updates are applied",
			testFunc: func(t *testing.T) {
				updatedSensor, err := fakeSensorClient.ArgoprojV1alpha1().Sensors(fakeSensor.Namespace).Get(fakeSensor.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.Equal(t, len(updatedSensor.Status.Nodes), 1)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.testFunc(t)
		})
	}
}

func TestIsDependencyResolved(t *testing.T) {
	fakeSensorClient := fakesensor.NewSimpleClientset()
	logger := common.NewArgoEventsLogger()
	fakeSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sensor",
			Namespace: "test",
		},
	}

	fakeSensor, err := fakeSensorClient.ArgoprojV1alpha1().Sensors(fakeSensor.Namespace).Create(fakeSensor)
	assert.Nil(t, err)

	dep1 := InitializeNode(fakeSensor, "dep1", v1alpha1.NodeTypeEventDependency, logger)
	assert.Equal(t, dep1.Phase, v1alpha1.NodePhaseNew)
	assert.NotNil(t, dep1.StartedAt)
	dep2 := InitializeNode(fakeSensor, "dep2", v1alpha1.NodeTypeEventDependency, logger)
	assert.Equal(t, dep1.Phase, v1alpha1.NodePhaseNew)
	assert.NotNil(t, dep1.StartedAt)

	dep1 = MarkUpdatedAt(fakeSensor, dep1.Name)
	assert.NotNil(t, dep1)
	dep2 = MarkUpdatedAt(fakeSensor, dep2.Name)
	assert.NotNil(t, dep2)

	resolved := IsDependencyResolved(fakeSensor, dep1.Name)
	assert.Equal(t, true, resolved)

	resolved = IsDependencyResolved(fakeSensor, dep2.Name)
	assert.Equal(t, true, resolved)

	dep3 := InitializeNode(fakeSensor, "dep3", v1alpha1.NodeTypeEventDependency, logger)
	MarkNodePhase(fakeSensor, dep3.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger)
	resolved = IsDependencyResolved(fakeSensor, dep3.Name)
	assert.Equal(t, true, resolved)
}

func TestMarkUpdatedAt(t *testing.T) {
	fakeSensorClient := fakesensor.NewSimpleClientset()
	logger := common.NewArgoEventsLogger()
	fakeSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sensor",
			Namespace: "test",
		},
	}

	fakeSensor, err := fakeSensorClient.ArgoprojV1alpha1().Sensors(fakeSensor.Namespace).Create(fakeSensor)
	assert.Nil(t, err)

	dep1 := InitializeNode(fakeSensor, "dep1", v1alpha1.NodeTypeEventDependency, logger)
	assert.Equal(t, dep1.Phase, v1alpha1.NodePhaseNew)
	assert.Empty(t, dep1.UpdatedAt.Time)

	dep1 = MarkUpdatedAt(fakeSensor, dep1.Name)
	assert.NotEmpty(t, dep1.UpdatedAt)
}

func TestMarkResolvedAt(t *testing.T) {
	fakeSensorClient := fakesensor.NewSimpleClientset()
	logger := common.NewArgoEventsLogger()
	fakeSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sensor",
			Namespace: "test",
		},
	}

	fakeSensor, err := fakeSensorClient.ArgoprojV1alpha1().Sensors(fakeSensor.Namespace).Create(fakeSensor)
	assert.Nil(t, err)

	dep1 := InitializeNode(fakeSensor, "dep1", v1alpha1.NodeTypeEventDependency, logger)
	assert.Equal(t, dep1.Phase, v1alpha1.NodePhaseNew)
	assert.Empty(t, dep1.ResolvedAt)

	dep1 = MarkResolvedAt(fakeSensor, dep1.Name)
	assert.NotEmpty(t, dep1.ResolvedAt)
}

func TestAreAllDependenciesResolved(t *testing.T) {
	fakeSensorClient := fakesensor.NewSimpleClientset()
	logger := common.NewArgoEventsLogger()
	fakeSensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sensor",
			Namespace: "test",
		},
	}

	fakeSensor, err := fakeSensorClient.ArgoprojV1alpha1().Sensors(fakeSensor.Namespace).Create(fakeSensor)
	assert.Nil(t, err)

	dep1 := InitializeNode(fakeSensor, "dep1", v1alpha1.NodeTypeEventDependency, logger)
	dep2 := InitializeNode(fakeSensor, "dep2", v1alpha1.NodeTypeEventDependency, logger)
	dep3 := InitializeNode(fakeSensor, "dep3", v1alpha1.NodeTypeEventDependency, logger)
	dep4 := InitializeNode(fakeSensor, "dep4", v1alpha1.NodeTypeEventDependency, logger)

	ok := AreAllDependenciesResolved(fakeSensor)
	assert.Equal(t, false, ok)

	MarkUpdatedAt(fakeSensor, dep1.Name)
	MarkUpdatedAt(fakeSensor, dep2.Name)
	ok = AreAllDependenciesResolved(fakeSensor)
	assert.Equal(t, false, ok)

	MarkUpdatedAt(fakeSensor, dep3.Name)
	MarkUpdatedAt(fakeSensor, dep4.Name)
	ok = AreAllDependenciesResolved(fakeSensor)
	assert.Equal(t, true, ok)
}
