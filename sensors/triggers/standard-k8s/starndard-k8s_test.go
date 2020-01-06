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

package standard_k8s

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicFake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
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

func TestFetchResource(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	sensorObj.Spec.Triggers[0].Template.Source = &v1alpha1.ArtifactLocation{
		Resource: deployment,
	}
	uObj, err := FetchResource(fake.NewSimpleClientset(), sensorObj, &sensorObj.Spec.Triggers[0])
	assert.Nil(t, err)
	assert.NotNil(t, uObj)
	assert.Equal(t, deployment.GetName(), uObj.GetName())
}

func TestExecute(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	sensorObj.Spec.Triggers[0].Template.Source = &v1alpha1.ArtifactLocation{
		Resource: deployment,
	}
	runtimeScheme := runtime.NewScheme()
	client := dynamicFake.NewSimpleDynamicClient(runtimeScheme)
	trigger := sensorObj.Spec.Triggers[0]
	namespacableClient := client.Resource(schema.GroupVersionResource{
		Resource: trigger.Template.GroupVersionResource.Resource,
		Version:  trigger.Template.GroupVersionResource.Version,
		Group:    trigger.Template.GroupVersionResource.Group,
	})
	uObj, err := Execute(sensorObj, deployment, namespacableClient, v1alpha1.Create)
	assert.Nil(t, err)
	assert.NotNil(t, uObj)
	assert.Equal(t, uObj.GetName(), deployment.GetName())
}
