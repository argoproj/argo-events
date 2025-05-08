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

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

func TestFilter(t *testing.T) {
	convey.Convey("Given a resource object, apply filter on it", t, func() {
		resourceEventSource := &v1alpha1.ResourceEventSource{
			GroupVersionResource: metav1.GroupVersionResource{
				Group:    "",
				Resource: "pods",
				Version:  "v1",
			},
			Filter: &v1alpha1.ResourceFilter{
				Labels: []v1alpha1.Selector{
					{
						Key:       "workflows.argoproj.io/phase",
						Operation: "==",
						Value:     "Succeeded",
					},
					{
						Key:       "name",
						Operation: "==",
						Value:     "my-workflow",
					},
				},
				Fields: []v1alpha1.Selector{
					{
						Key:       "metadata.name",
						Operation: "==",
						Value:     "fak*",
					},
					{
						Key:       "status.phase",
						Operation: "!=",
						Value:     "Error",
					},
					{
						Key:       "spec.serviceAccountName",
						Operation: "=",
						Value:     "test*",
					},
				},
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake",
				Namespace: "fake",
				Labels: map[string]string{
					"workflows.argoproj.io/phase": "Succeeded",
					"name":                        "my-workflow",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-sa",
			},
			Status: corev1.PodStatus{
				Phase: "Running",
			},
		}
		pod, err := fake.NewSimpleClientset().CoreV1().Pods("fake").Create(context.TODO(), pod, metav1.CreateOptions{})
		convey.So(err, convey.ShouldBeNil)

		outmap := make(map[string]interface{})
		jsonData, err := json.Marshal(pod)
		convey.So(err, convey.ShouldBeNil)
		err = json.Unmarshal(jsonData, &outmap)
		convey.So(err, convey.ShouldBeNil)

		pass := passFilters(&InformerEvent{
			Obj:  &unstructured.Unstructured{Object: outmap},
			Type: "ADD",
		}, resourceEventSource.Filter, time.Now(), logging.NewArgoEventsLogger())
		convey.So(pass, convey.ShouldBeTrue)
	})
}

func TestLabelSelector(t *testing.T) {
	// Test equality operators =, == and in
	for _, op := range []string{"==", "=", "in"} {
		t.Run(fmt.Sprintf("Test operator %v", op), func(t *testing.T) {
			r, err := LabelSelector([]v1alpha1.Selector{{
				Key:       "key",
				Operation: op,
				Value:     "1",
			}})
			if err != nil {
				t.Fatal(err)
			}
			validL := &labels.Set{"key": "1"}
			if !r.Matches(validL) {
				t.Errorf("didnot match %v", validL)
			}
			invalidL := &labels.Set{"key": "2"}
			if r.Matches(invalidL) {
				t.Errorf("matched %v", invalidL)
			}
		})
	}
	// Test inequality operators != and notin
	for _, op := range []string{"!=", "notin"} {
		t.Run(fmt.Sprintf("Test operator %v", op), func(t *testing.T) {
			r, err := LabelSelector([]v1alpha1.Selector{{
				Key:       "key",
				Operation: op,
				Value:     "1",
			}})
			if err != nil {
				t.Fatal(err)
			}
			validL := &labels.Set{"key": "2"}
			if !r.Matches(validL) {
				t.Errorf("didnot match %v", validL)
			}
			invalidL := &labels.Set{"key": "1"}
			if r.Matches(invalidL) {
				t.Errorf("matched %v", invalidL)
			}
		})
	}
	// Test greater than operator
	t.Run("Test operator gt", func(t *testing.T) {
		r, err := LabelSelector([]v1alpha1.Selector{{
			Key:       "key",
			Operation: "gt",
			Value:     "1",
		}})
		if err != nil {
			t.Fatal(err)
		}
		validL := &labels.Set{"key": "2"}
		if !r.Matches(validL) {
			t.Errorf("didnot match %v", validL)
		}
		invalidL := &labels.Set{"key": "1"}
		if r.Matches(invalidL) {
			t.Errorf("matched %v", invalidL)
		}
	})
	// Test lower than operator
	t.Run("Test operator lt", func(t *testing.T) {
		r, err := LabelSelector([]v1alpha1.Selector{{
			Key:       "key",
			Operation: "lt",
			Value:     "2",
		}})
		if err != nil {
			t.Fatal(err)
		}
		validL := &labels.Set{"key": "1"}
		if !r.Matches(validL) {
			t.Errorf("didnot match %v", validL)
		}
		invalidL := &labels.Set{"key": "2"}
		if r.Matches(invalidL) {
			t.Errorf("matched %v", invalidL)
		}
	})
	// Test exists operator
	t.Run("Test operator exists", func(t *testing.T) {
		r, err := LabelSelector([]v1alpha1.Selector{{
			Key:       "key",
			Operation: "exists",
		}})
		if err != nil {
			t.Fatal(err)
		}
		validL := &labels.Set{"key": "something"}
		if !r.Matches(validL) {
			t.Errorf("didnot match %v", validL)
		}
		invalidL := &labels.Set{"notkey": "something"}
		if r.Matches(invalidL) {
			t.Errorf("matched %v", invalidL)
		}
	})
	// Test does not exist operator
	t.Run("Test operator !", func(t *testing.T) {
		r, err := LabelSelector([]v1alpha1.Selector{{
			Key:       "key",
			Operation: "!",
		}})
		if err != nil {
			t.Fatal(err)
		}
		validL := &labels.Set{"notkey": "something"}
		if !r.Matches(validL) {
			t.Errorf("didnot match %v", validL)
		}
		invalidL := &labels.Set{"key": "something"}
		if r.Matches(invalidL) {
			t.Errorf("matched %v", invalidL)
		}
	})
	// Test default operator
	t.Run("Test default operator", func(t *testing.T) {
		r, err := LabelSelector([]v1alpha1.Selector{{
			Key:       "key",
			Operation: "",
			Value:     "something",
		}})
		if err != nil {
			t.Fatal(err)
		}
		validL := &labels.Set{"key": "something"}
		if !r.Matches(validL) {
			t.Errorf("didnot match %v", validL)
		}
		invalidL := &labels.Set{"key": "not something"}
		if r.Matches(invalidL) {
			t.Errorf("matched %v", invalidL)
		}
	})
	// Test invalid operators <= and >=
	for _, op := range []string{"<=", ">="} {
		t.Run(fmt.Sprintf("Invalid operator %v", op), func(t *testing.T) {
			_, err := LabelSelector([]v1alpha1.Selector{{
				Key:       "workflows.argoproj.io/phase",
				Operation: op,
				Value:     "1",
			}})
			if err == nil {
				t.Errorf("Invalid operator should throw error")
			}
		})
	}
	// Test comma separated values for in
	t.Run("Comma separated values", func(t *testing.T) {
		r, err := LabelSelector([]v1alpha1.Selector{{
			Key:       "key",
			Operation: "in",
			Value:     "a,b,",
		}})
		if err != nil {
			t.Fatal("valid value threw error, value %w", err)
		}
		for _, validL := range []labels.Set{{"key": "a"}, {"key": "b"}, {"key": ""}} {
			if !r.Matches(validL) {
				t.Errorf("didnot match %v", validL)
			}
		}
		invalidL := &labels.Set{"key": "c"}
		if r.Matches(invalidL) {
			t.Errorf("matched %v", invalidL)
		}
	})
}
