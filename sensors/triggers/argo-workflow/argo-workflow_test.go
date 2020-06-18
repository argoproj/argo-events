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
package argo_workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicFake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
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
					K8s: &v1alpha1.StandardK8STrigger{
						GroupVersionResource: metav1.GroupVersionResource{
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

func getFakeWfTrigger() *ArgoWorkflowTrigger {
	runtimeScheme := runtime.NewScheme()
	client := dynamicFake.NewSimpleDynamicClient(runtimeScheme)
	un := newUnstructured("argoproj.io/v1alpha1", "Workflow", "fake", "test")
	artifact := apicommon.NewResource(un)
	trigger := &v1alpha1.Trigger{
		Template: &v1alpha1.TriggerTemplate{
			Name: "fake",
			ArgoWorkflow: &v1alpha1.ArgoWorkflowTrigger{
				Source: &v1alpha1.ArtifactLocation{
					Resource: &artifact,
				},
				Operation: "Submit",
				GroupVersionResource: metav1.GroupVersionResource{
					Group:    "argoproj.io",
					Version:  "v1alpha1",
					Resource: "workflows",
				},
			},
		},
	}
	return NewArgoWorkflowTrigger(fake.NewSimpleClientset(), client, sensorObj.DeepCopy(), trigger, common.NewArgoEventsLogger())
}

func TestFetchResource(t *testing.T) {
	trigger := getFakeWfTrigger()
	resource, err := trigger.FetchResource()
	assert.Nil(t, err)
	assert.NotNil(t, resource)
	obj, ok := resource.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, "test", obj.GetName())
	assert.Equal(t, "Workflow", obj.GroupVersionKind().Kind)
}

func TestApplyResourceParameters(t *testing.T) {

}
