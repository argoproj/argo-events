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

package triggers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
)

var sensorObj = &v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor",
		Namespace: "fake",
	},
	Spec: v1alpha1.SensorSpec{
		Triggers: []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s: &v1alpha1.StandardK8sTrigger{
						GroupVersionResource: &metav1.GroupVersionResource{
							Group:    "apps",
							Version:  "v1",
							Resource: "deployments",
						},
					},
				},
			},
		},
	},
}

func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
				"labels": map[string]interface{}{
					"name": name,
				},
			},
		},
	}
}

func TestApplySwitches(t *testing.T) {
	obj := sensorObj.DeepCopy()

	tests := []struct {
		name           string
		templateSwitch *v1alpha1.TriggerSwitch
		updateFunc     func()
		result         bool
	}{
		{
			name:           "no switches",
			templateSwitch: nil,
			updateFunc:     func() {},
			result:         true,
		},
		{
			name: "success: apply any switch",
			templateSwitch: &v1alpha1.TriggerSwitch{
				Any: []string{
					"group-1",
					"group-2",
				},
			},
			updateFunc: func() {
				groupId1 := obj.NodeID("group-1")
				groupId2 := obj.NodeID("group-2")
				obj.Status = v1alpha1.SensorStatus{
					Nodes: map[string]v1alpha1.NodeStatus{
						groupId1: {
							Type:  v1alpha1.NodeTypeDependencyGroup,
							Phase: v1alpha1.NodePhaseComplete,
							ID:    groupId1,
						},
						groupId2: {
							Type:  v1alpha1.NodeTypeDependencyGroup,
							Phase: v1alpha1.NodePhaseNew,
							ID:    groupId2,
						},
					},
				}
			},
			result: true,
		},
		{
			name: "failure: apply any switch",
			templateSwitch: &v1alpha1.TriggerSwitch{
				Any: []string{
					"group-1",
					"group-2",
				},
			},
			updateFunc: func() {
				groupId1 := obj.NodeID("group-1")
				groupId2 := obj.NodeID("group-2")
				obj.Status = v1alpha1.SensorStatus{
					Nodes: map[string]v1alpha1.NodeStatus{
						groupId1: {
							Type:  v1alpha1.NodeTypeDependencyGroup,
							Phase: v1alpha1.NodePhaseError,
							ID:    groupId1,
						},
						groupId2: {
							Type:  v1alpha1.NodeTypeDependencyGroup,
							Phase: v1alpha1.NodePhaseNew,
							ID:    groupId2,
						},
					},
				}
			},
			result: false,
		},
		{
			name: "success: apply all switch",
			templateSwitch: &v1alpha1.TriggerSwitch{
				All: []string{
					"group-1",
					"group-2",
				},
			},
			updateFunc: func() {
				groupId1 := obj.NodeID("group-1")
				groupId2 := obj.NodeID("group-2")
				obj.Status = v1alpha1.SensorStatus{
					Nodes: map[string]v1alpha1.NodeStatus{
						groupId1: {
							Type:  v1alpha1.NodeTypeDependencyGroup,
							Phase: v1alpha1.NodePhaseComplete,
							ID:    groupId1,
						},
						groupId2: {
							Type:  v1alpha1.NodeTypeDependencyGroup,
							Phase: v1alpha1.NodePhaseComplete,
							ID:    groupId2,
						},
					},
				}
			},
			result: true,
		},
		{
			name: "failure: apply all switch",
			templateSwitch: &v1alpha1.TriggerSwitch{
				All: []string{
					"group-1",
					"group-2",
				},
			},
			updateFunc: func() {
				groupId1 := obj.NodeID("group-1")
				groupId2 := obj.NodeID("group-2")
				obj.Status = v1alpha1.SensorStatus{
					Nodes: map[string]v1alpha1.NodeStatus{
						groupId1: {
							Type:  v1alpha1.NodeTypeDependencyGroup,
							Phase: v1alpha1.NodePhaseError,
							ID:    groupId1,
						},
						groupId2: {
							Type:  v1alpha1.NodeTypeDependencyGroup,
							Phase: v1alpha1.NodePhaseComplete,
							ID:    groupId2,
						},
					},
				}
			},
			result: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obj.Spec.Triggers[0].Template.Switch = test.templateSwitch
			test.updateFunc()
			result := ApplySwitches(obj, &obj.Spec.Triggers[0])
			assert.Equal(t, test.result, result)
		})
	}
}
