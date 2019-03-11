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
	"github.com/mitchellh/mapstructure"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

var (
	ese          = &ResourceEventSourceExecutor{}
	resourceList = metav1.APIResourceList{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{
				Group:        "",
				Version:      "v1",
				Name:         "fake-1",
				Kind:         "Fake",
				Namespaced:   true,
				ShortNames:   []string{"f1"},
				SingularName: "fake-1",
				Verbs:        []string{"watch"},
			},
		},
	}
)

func TestServerResourceForGVK(t *testing.T) {
	convey.Convey("Given a resource gvk, discover the resource", t, func() {
		result, err := ese.serverResourceForGVK(&resourceList, "Fake")
		convey.So(err, convey.ShouldBeNil)
		convey.So(result.Kind, convey.ShouldEqual, "Fake")
	})
}

func TestCanWatchResource(t *testing.T) {
	convey.Convey("Given a resource, test if it can watched", t, func() {
		result, err := ese.serverResourceForGVK(&resourceList, "Fake")
		convey.So(err, convey.ShouldBeNil)
		ok := ese.canWatchResource(result)
		convey.So(ok, convey.ShouldEqual, true)
	})
}

func TestResolveGroupVersion(t *testing.T) {
	convey.Convey("Given a resource, resolve the group and version", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		gv := ese.resolveGroupVersion(ps.(*resource))
		convey.So(gv, convey.ShouldEqual, "v1")
		gv = ese.resolveGroupVersion(&resource{
			GroupVersionKind: metav1.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Fake",
				Version: "v1alpha1",
			},
			Version: "v1alpha1",
		})
		convey.So(gv, convey.ShouldEqual, "argoproj.io/v1alpha1")
	})
}

func TestFilter(t *testing.T) {
	convey.Convey("Given a resource object, apply filter on it", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake",
				Namespace: "fake",
				Labels: map[string]string{
					"workflows.argoproj.io/phase": "Succeeded",
					"name": "my-workflow",
				},
			},
		}
		pod, err = fake.NewSimpleClientset().CoreV1().Pods("fake").Create(pod)
		convey.So(err, convey.ShouldBeNil)

		outmap := make(map[string]interface{})
		err = mapstructure.Decode(pod, &outmap)
		convey.So(err, convey.ShouldBeNil)

		ok := ese.passFilters("fake", &unstructured.Unstructured{
			Object: outmap,
		}, ps.(*resource).Filter)
		convey.So(ok, convey.ShouldEqual, true)
	})
}
