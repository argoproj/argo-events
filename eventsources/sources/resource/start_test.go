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

package resource

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

func TestFilter(t *testing.T) {
	convey.Convey("Given a resource object, apply filter on it", t, func() {
		resourceEventSource := &v1alpha1.ResourceEventSource{
			Namespace: "fake",
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
		pod, err := fake.NewSimpleClientset().CoreV1().Pods("fake").Create(context.Background(), pod, metav1.CreateOptions{})
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
