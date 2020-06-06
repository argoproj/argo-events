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
	"testing"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/mitchellh/mapstructure"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
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
		}
		pod, err := fake.NewSimpleClientset().CoreV1().Pods("fake").Create(pod)
		convey.So(err, convey.ShouldBeNil)

		outmap := make(map[string]interface{})
		err = mapstructure.Decode(pod, &outmap)
		convey.So(err, convey.ShouldBeNil)

		pass := passFilters(&InformerEvent{
			Obj:  &unstructured.Unstructured{Object: outmap},
			Type: "ADD",
		}, resourceEventSource.Filter, time.Now())
		convey.So(pass, convey.ShouldBeTrue)
	})
}
