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

package policy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/fake"

	"github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
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

func TestResourceLabels_ApplyPolicy(t *testing.T) {
	uObj := newUnstructured("apps/v1", "Deployment", "fake", "test")
	runtimeScheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(runtimeScheme, uObj)
	artifact := common.NewResource(uObj)
	jitter := common.NewAmount("0.5")
	trigger := &v1alpha1.Trigger{
		Template: &v1alpha1.TriggerTemplate{
			Name: "fake-trigger",
			K8s: &v1alpha1.StandardK8STrigger{
				GroupVersionResource: metav1.GroupVersionResource{
					Group:    "apps",
					Resource: "deployments",
					Version:  "v1",
				},
				Source: &v1alpha1.ArtifactLocation{
					Resource: &artifact,
				},
			},
		},
		Policy: &v1alpha1.TriggerPolicy{
			K8s: &v1alpha1.K8SResourcePolicy{
				ErrorOnBackoffTimeout: true,
				Labels: map[string]string{
					"complete": "true",
				},
				Backoff: common.Backoff{
					Steps:    2,
					Duration: time.Second * 1,
					Factor:   common.NewAmount("2"),
					Jitter:   &jitter,
				},
			},
		},
	}

	namespacableClient := client.Resource(schema.GroupVersionResource{
		Resource: trigger.Template.K8s.GroupVersionResource.Resource,
		Version:  trigger.Template.K8s.GroupVersionResource.Version,
		Group:    trigger.Template.K8s.GroupVersionResource.Group,
	})

	tests := []struct {
		name       string
		updateFunc func(deployment *unstructured.Unstructured) (*unstructured.Unstructured, error)
		testFunc   func(err error)
	}{
		{
			name: "success",
			updateFunc: func(deployment *unstructured.Unstructured) (*unstructured.Unstructured, error) {
				labels := deployment.GetLabels()
				if labels == nil {
					labels = map[string]string{}
				}
				labels["complete"] = "true"
				deployment.SetLabels(labels)
				return namespacableClient.Namespace("fake").Update(deployment, metav1.UpdateOptions{})
			},
			testFunc: func(err error) {
				assert.Nil(t, err)
			},
		},
		{
			name: "failure",
			updateFunc: func(deployment *unstructured.Unstructured) (i *unstructured.Unstructured, e error) {
				labels := deployment.GetLabels()
				if labels == nil {
					labels = map[string]string{}
				}
				labels["complete"] = "false"
				deployment.SetLabels(labels)
				return namespacableClient.Namespace("fake").Update(deployment, metav1.UpdateOptions{})
			},
			testFunc: func(err error) {
				assert.NotNil(t, err)
				assert.Equal(t, wait.ErrWaitTimeout.Error(), err.Error())
			},
		},
	}

	resourceLabelsPolicy := &ResourceLabels{
		Obj:     uObj,
		Trigger: trigger,
		Client:  namespacableClient,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			uObj, err = test.updateFunc(uObj)
			assert.Nil(t, err)
			err = resourceLabelsPolicy.ApplyPolicy()
			test.testFunc(err)
		})
	}
}
