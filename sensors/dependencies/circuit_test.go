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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

var sensorObj = v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor",
		Namespace: "fake-namespace",
	},
	Spec: v1alpha1.SensorSpec{
		Dependencies: []v1alpha1.EventDependency{
			{
				Name:        "dep-1",
				EventName:   "example-1",
				GatewayName: "fake-gateway",
			},
			{
				Name:        "dep-2",
				EventName:   "example-2",
				GatewayName: "fake-gateway",
			},
			{
				Name:        "dep-3",
				EventName:   "example-3",
				GatewayName: "fake-gateway",
			},
			{
				Name:        "dep-4",
				EventName:   "example-4",
				GatewayName: "fake-gateway",
			},
		},
		DependencyGroups: []v1alpha1.DependencyGroup{
			{
				Name: "group1",
				Dependencies: []string{
					"dep-1",
				},
			},
			{
				Name: "group2",
				Dependencies: []string{
					"dep-2",
				},
			},
			{
				Name: "group3",
				Dependencies: []string{
					"dep-3",
					"dep-4",
				},
			},
		},
	},
}

func TestResolveCircuit(t *testing.T) {
	logger := common.NewArgoEventsLogger()
	obj := sensorObj.DeepCopy()

	for _, dependency := range obj.Spec.Dependencies {
		snctrl.InitializeNode(obj, dependency.Name, v1alpha1.NodeTypeEventDependency, logger)
	}
	for _, group := range obj.Spec.DependencyGroups {
		snctrl.InitializeNode(obj, group.Name, v1alpha1.NodeTypeDependencyGroup, logger)
	}

	tests := []struct {
		name       string
		result     bool
		updateFunc func()
		snapshot   []string
	}{
		{
			name:   "apply false OR logic",
			result: false,
			updateFunc: func() {
				obj.Spec.Circuit = "group1 || group2 || group3"
			},
			snapshot: nil,
		},
		{
			name:   "apply OR logic",
			result: true,
			updateFunc: func() {
				obj.Spec.Circuit = "group1 || group2 || group3"
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[0].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger, "dependency is complete")
			},
			snapshot: []string{"dep-1"},
		},
		{
			name:   "apply false AND logic",
			result: false,
			updateFunc: func() {
				obj.Spec.Circuit = "group1 && (group2 || group3)"
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[0].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger, "dependency is complete")
			},
			snapshot: nil,
		},
		{
			name:   "apply AND logic",
			result: true,
			updateFunc: func() {
				obj.Spec.Circuit = "group1 && group2 && group3"
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[0].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger, "dependency is complete")
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[1].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger, "dependency is complete")
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[2].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger, "dependency is complete")
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[3].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger, "dependency is complete")
			},
			snapshot: []string{"dep-3", "dep-4", "dep-1", "dep-2"},
		},
		{
			name:   "apply NOT logic",
			result: true,
			updateFunc: func() {
				obj.Spec.Circuit = "!group1 && !group2"
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[0].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseNew, nil, logger, "dependency is init")
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[1].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseNew, nil, logger, "dependency is init")
			},
			snapshot: []string{"dep-3", "dep-4"},
		},
		{
			name:   "group with multiple dependencies is complete",
			result: true,
			updateFunc: func() {
				obj.Spec.Circuit = "(group1 && group2) || group3"
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[0].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseNew, nil, logger, "dependency is init")
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[1].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseNew, nil, logger, "dependency is init")
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[2].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger, "dependency is complete")
				snctrl.MarkNodePhase(obj, obj.Spec.Dependencies[3].Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, nil, logger, "dependency is complete")
			},
			snapshot: []string{"dep-3", "dep-4"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.updateFunc()
			ok, snapshot, err := ResolveCircuit(obj, logger)
			assert.Nil(t, err)
			assert.ElementsMatch(t, test.snapshot, snapshot)
			assert.Equal(t, test.result, ok)
		})
	}
}
