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

package standard_k8s

import (
	"context"
	"testing"
	"time"

	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicFake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	k8stesting "k8s.io/client-go/testing"
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
					K8s:  &v1alpha1.StandardK8STrigger{},
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
			"spec": map[string]interface{}{
				"replica": "1",
			},
		},
	}
}

func TestStandardK8sTrigger_FetchResource(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	artifact := v1alpha1.NewK8SResource(deployment)
	sensorObj.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: &artifact,
	}
	runtimeScheme := runtime.NewScheme()
	client := dynamicFake.NewSimpleDynamicClient(runtimeScheme)
	impl := NewStandardK8sTrigger(fake.NewSimpleClientset(), client, sensorObj, &sensorObj.Spec.Triggers[0], logging.NewArgoEventsLogger())
	resource, err := impl.FetchResource(context.TODO())
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	uObj, ok := resource.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, deployment.GetName(), uObj.GetName())
}

func TestStandardK8sTrigger_ApplyResourceParameters(t *testing.T) {
	fakeSensor := sensorObj.DeepCopy()

	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			DataContentType: v1alpha1.MediaTypeJSON,
			Subject:         "example-1",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			Type:            "webhook",
			ID:              "1",
			Time:            metav1.Time{Time: time.Now().UTC()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"} }"),
	}

	testEvents := map[string]*v1alpha1.Event{
		"dep-1": event,
	}

	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	artifact := v1alpha1.NewK8SResource(deployment)
	fakeSensor.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: &artifact,
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
	impl := NewStandardK8sTrigger(fake.NewSimpleClientset(), client, fakeSensor, &fakeSensor.Spec.Triggers[0], logging.NewArgoEventsLogger())

	out, err := impl.ApplyResourceParameters(testEvents, deployment)
	assert.Nil(t, err)

	updatedObj, ok := out.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, "faketest", updatedObj.GetName())
}

func TestStandardK8sTrigger_Execute(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	artifact := v1alpha1.NewK8SResource(deployment)
	sensorObj.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: &artifact,
	}
	runtimeScheme := runtime.NewScheme()
	client := dynamicFake.NewSimpleDynamicClient(runtimeScheme)
	impl := NewStandardK8sTrigger(fake.NewSimpleClientset(), client, sensorObj, &sensorObj.Spec.Triggers[0], logging.NewArgoEventsLogger())
	ctx := context.TODO()

	resource, err := impl.Execute(ctx, nil, deployment)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	uObj, ok := resource.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, deployment.GetName(), uObj.GetName())

	labels := uObj.GetLabels()
	labels["update"] = "ok"
	uObj.SetLabels(labels)

	sensorObj.Spec.Triggers[0].Template.K8s.Operation = v1alpha1.Update
	impl = NewStandardK8sTrigger(fake.NewSimpleClientset(), client, sensorObj, &sensorObj.Spec.Triggers[0], logging.NewArgoEventsLogger())
	resource, err = impl.Execute(ctx, nil, uObj)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	uObj, ok = resource.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, deployment.GetName(), uObj.GetName())
	assert.Equal(t, "ok", uObj.GetLabels()["update"])

	labels = uObj.GetLabels()
	labels["foo"] = "bar"
	uObj.SetLabels(labels)

	sensorObj.Spec.Triggers[0].Template.K8s.Operation = v1alpha1.Patch
	sensorObj.Spec.Triggers[0].Template.K8s.PatchStrategy = k8stypes.MergePatchType

	impl = NewStandardK8sTrigger(fake.NewSimpleClientset(), client, sensorObj, &sensorObj.Spec.Triggers[0], logging.NewArgoEventsLogger())
	resource, err = impl.Execute(ctx, nil, uObj)
	assert.Nil(t, err)
	assert.NotNil(t, resource)
	uObj, ok = resource.(*unstructured.Unstructured)
	assert.Equal(t, true, ok)
	assert.Equal(t, "bar", uObj.GetLabels()["foo"])

	deleted := false

	sensorObj.Spec.Triggers[0].Template.K8s.Operation = v1alpha1.Delete

	client.PrependReactor("delete", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		deleteAction := action.(k8stesting.DeleteAction)
		if deleteAction.GetName() == deployment.GetName() && deleteAction.GetNamespace() == deployment.GetNamespace() {
			deleted = true
		}
		return true, nil, nil
	})

	impl = NewStandardK8sTrigger(fake.NewSimpleClientset(), client, sensorObj, &sensorObj.Spec.Triggers[0], logging.NewArgoEventsLogger())
	resource, err = impl.Execute(ctx, nil, uObj)
	assert.Nil(t, err)
	assert.Nil(t, resource)
	assert.True(t, deleted)
}
