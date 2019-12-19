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

package sensors

import (
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorFake "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	"github.com/argoproj/argo-events/sensors/types"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	registry = runtime.NewEquivalentResourceRegistry()
)

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
					GroupVersionResource: &metav1.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					},
				},
			},
		},
	},
}

func TestIsEligibleForExecution(t *testing.T) {
	obj := sensorObj.DeepCopy()
	group1 := obj.NodeID("group1")
	dep1 := obj.NodeID("dep1")
	obj.Spec.DependencyGroups = []v1alpha1.DependencyGroup{
		{
			Name: "group1",
			Dependencies: []string{
				"dep1",
			},
		},
	}
	obj.Spec.Dependencies = []v1alpha1.EventDependency{
		{
			Name:        "dep1",
			GatewayName: "webhook-gateway",
			EventName:   "example-1",
		},
	}
	obj.Status.Nodes = map[string]v1alpha1.NodeStatus{
		group1: {
			ID:          group1,
			Name:        "group1",
			DisplayName: "group1",
			Type:        v1alpha1.NodeTypeDependencyGroup,
		},
		dep1: {
			ID:          dep1,
			Name:        "dep1",
			DisplayName: "dep1",
			Type:        v1alpha1.NodeTypeEventDependency,
		},
	}

	tests := []struct {
		name               string
		dependencyStatus   v1alpha1.NodePhase
		errOnFailedRound   bool
		triggerCycleStatus v1alpha1.TriggerCycleState
		hasError           bool
		result             bool
		circuit            string
		updateFunc         func()
	}{
		{
			name:     "if error on failed round and trigger cycle in failed state",
			hasError: true,
			result:   false,
			updateFunc: func() {
				obj.Status.TriggerCycleStatus = v1alpha1.TriggerCycleFailure
				obj.Spec.ErrorOnFailedRound = true
			},
		},
		{
			name: "if the circuit logic is complete",
			updateFunc: func() {
				obj.Spec.Circuit = "group1"
				node := obj.Status.Nodes[dep1]
				node.Phase = v1alpha1.NodePhaseComplete
				obj.Status.Nodes[dep1] = node
				obj.Status.TriggerCycleStatus = ""
				obj.Spec.ErrorOnFailedRound = true
			},
			hasError: false,
			result:   true,
		},
		{
			name: "if all dependencies are complete",
			updateFunc: func() {
				obj.Spec.Circuit = ""
				node := obj.Status.Nodes[dep1]
				node.Phase = v1alpha1.NodePhaseComplete
				obj.Status.Nodes[dep1] = node
			},

			hasError: false,
			result:   true,
		},
		{
			name: "if no dependencies are complete",
			updateFunc: func() {
				obj.Spec.Circuit = ""
				node := obj.Status.Nodes[dep1]
				node.Phase = v1alpha1.NodePhaseActive
				obj.Status.Nodes[dep1] = node
			},

			hasError: false,
			result:   false,
		},
	}

	logger := common.NewArgoEventsLogger()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.updateFunc()
			result, err := isEligibleForExecution(obj, logger)
			if test.hasError {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, test.result, result)
		})
	}
}

func TestOperateEventNotification(t *testing.T) {
	sensorClient := sensorFake.NewSimpleClientset()
	dynamicClient := dfake.NewSimpleDynamicClient(runtime.NewScheme())
	k8sClient := fake.NewSimpleClientset()
	obj := sensorObj.DeepCopy()
	sensorCtx := NewSensorContext(sensorClient, k8sClient, dynamicClient, obj, "1")

	event := &apicommon.Event{
		Context: apicommon.EventContext{
			DataContentType: "application/json",
			Subject:         "example-1",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			Type:            "webhook",
			ID:              "1",
			Time:            metav1.MicroTime{Time: time.Now()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"} }"),
	}

	group1 := obj.NodeID("group1")
	dep1 := obj.NodeID("dep1")
	obj.Spec.DependencyGroups = []v1alpha1.DependencyGroup{
		{
			Name: "group1",
			Dependencies: []string{
				"dep1",
			},
		},
	}
	obj.Spec.Dependencies = []v1alpha1.EventDependency{
		{
			Name:        "dep1",
			GatewayName: "webhook-gateway",
			EventName:   "example-1",
		},
	}
	obj.Status.Nodes = map[string]v1alpha1.NodeStatus{
		group1: {
			ID:          group1,
			Name:        "group1",
			DisplayName: "group1",
			Type:        v1alpha1.NodeTypeDependencyGroup,
		},
		dep1: {
			ID:          dep1,
			Name:        "dep1",
			DisplayName: "dep1",
			Type:        v1alpha1.NodeTypeEventDependency,
			Phase:       v1alpha1.NodePhaseActive,
		},
	}

	deployment := newUnstructured("apps/v1", "Deployment", "fake", "fake-deployment")
	obj.Spec.Triggers[0].Template.Source = &v1alpha1.ArtifactLocation{
		Resource: deployment,
	}

	err := sensorCtx.operateEventNotification(&types.Notification{
		Event:            event,
		EventDependency:  &obj.Spec.Dependencies[0],
		Sensor:           obj,
		NotificationType: v1alpha1.EventNotification,
	})
	assert.Nil(t, err)

	assert.Equal(t, v1alpha1.NodePhaseComplete, obj.Status.Nodes[dep1].Phase)

	nsClient := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	})
	newDeployment, err := nsClient.Namespace("fake").Get("fake-deployment", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, deployment.GetName(), newDeployment.GetName())

	// Delete the deployment for later test cases
	err = nsClient.Namespace("fake").Delete("fake-deployment", &metav1.DeleteOptions{})
	assert.Nil(t, err)

	// Mark all dependency nodes as active
	for _, dependency := range sensorCtx.Sensor.Spec.Dependencies {
		snctrl.MarkNodePhase(sensorCtx.Sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, sensorCtx.Logger, "dependency is re-activated")
	}
	// Mark all dependency groups as active
	for _, group := range sensorCtx.Sensor.Spec.DependencyGroups {
		snctrl.MarkNodePhase(sensorCtx.Sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, sensorCtx.Logger, "dependency group is re-activated")
	}

	// Apply filters that fail
	obj.Spec.Dependencies[0].Filters = &v1alpha1.EventDependencyFilter{
		Name: "data-filter",
		Data: []v1alpha1.DataFilter{
			{
				Path: "name.first",
				Type: "string",
				Value: []string{
					"not-fake",
				},
			},
		},
	}

	err = sensorCtx.operateEventNotification(&types.Notification{
		Event:            event,
		EventDependency:  &obj.Spec.Dependencies[0],
		Sensor:           obj,
		NotificationType: v1alpha1.EventNotification,
	})
	assert.NotNil(t, err)
	assert.Equal(t, v1alpha1.NodePhaseError, obj.Status.Nodes[dep1].Phase)
}
