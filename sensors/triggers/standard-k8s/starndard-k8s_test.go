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
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

func TestStandardK8sTrigger_FetchResource(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	sensorObj.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: deployment,
	}
	runtimeScheme := runtime.NewScheme()
	client := dynamicFake.NewSimpleDynamicClient(runtimeScheme)
	impl := NewStandardK8sTrigger(fake.NewSimpleClientset(), client, sensorObj, &sensorObj.Spec.Triggers[0], common.NewArgoEventsLogger())
	resource, err := impl.FetchResource()
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	uObj, ok := resource.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, deployment.GetName(), uObj.GetName())
}

func TestStandardK8sTrigger_ApplyResourceParameters(t *testing.T) {
	fakeSensor := sensorObj.DeepCopy()

	event := apicommon.Event{
		Context: apicommon.EventContext{
			DataContentType: common.MediaTypeJSON,
			Subject:         "example-1",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			Type:            "webhook",
			ID:              "1",
			Time:            metav1.MicroTime{Time: time.Now()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"} }"),
	}

	dep := fakeSensor.NodeID("dep-1")

	fakeSensor.Status = v1alpha1.SensorStatus{
		Nodes: map[string]v1alpha1.NodeStatus{
			dep: {
				ID:          dep,
				Name:        dep,
				DisplayName: dep,
				Type:        v1alpha1.NodeTypeEventDependency,
				Event:       &event,
			},
		},
	}

	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	fakeSensor.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: deployment,
	}

	fakeSensor.Spec.Triggers[0].Template.K8s.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "dep-1",
				DataKey:        "name.first",
			},
			Dest:      "metadata.name",
			Operation: "prepend",
		},
	}

	runtimeScheme := runtime.NewScheme()
	client := dynamicFake.NewSimpleDynamicClient(runtimeScheme)
	impl := NewStandardK8sTrigger(fake.NewSimpleClientset(), client, fakeSensor, &fakeSensor.Spec.Triggers[0], common.NewArgoEventsLogger())

	out, err := impl.ApplyResourceParameters(fakeSensor, deployment)
	assert.Nil(t, err)

	updatedObj, ok := out.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, "faketest", updatedObj.GetName())
}

func TestStandardK8sTrigger_Execute(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	sensorObj.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: deployment,
	}
	runtimeScheme := runtime.NewScheme()
	client := dynamicFake.NewSimpleDynamicClient(runtimeScheme)
	impl := NewStandardK8sTrigger(fake.NewSimpleClientset(), client, sensorObj, &sensorObj.Spec.Triggers[0], common.NewArgoEventsLogger())

	resource, err := impl.Execute(deployment)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	uObj, ok := resource.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, deployment.GetName(), uObj.GetName())

	sensorObj.Spec.Triggers[0].Template.K8s.Operation = v1alpha1.Update
	impl = NewStandardK8sTrigger(fake.NewSimpleClientset(), client, sensorObj, &sensorObj.Spec.Triggers[0], common.NewArgoEventsLogger())
	resource, err = impl.Execute(deployment)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	uObj, ok = resource.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, deployment.GetName(), uObj.GetName())
}
