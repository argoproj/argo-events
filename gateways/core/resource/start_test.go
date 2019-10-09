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
					"name":                        "my-workflow",
				},
			},
		}
		pod, err = fake.NewSimpleClientset().CoreV1().Pods("fake").Create(pod)
		convey.So(err, convey.ShouldBeNil)

		outmap := make(map[string]interface{})
		err = mapstructure.Decode(pod, &outmap)
		convey.So(err, convey.ShouldBeNil)

		err = passFilters(&unstructured.Unstructured{
			Object: outmap,
		}, ps.(*resource).Filter)
		convey.So(err, convey.ShouldBeNil)
	})
}
