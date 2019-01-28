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

package common

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGenerateK8sEvent(t *testing.T) {
	convey.Convey("Given an event", t, func() {
		fakeclientset := fake.NewSimpleClientset()

		convey.Convey("Generate a K8s event", func() {

			convey.So(GenerateK8sEvent(fakeclientset, "fake event", StateChangeEventType,
				"state change", "fake-component", "fake-namespace", "1", "fake-kind",
				map[string]string{"fake": "fake"}), convey.ShouldBeEmpty)

			convey.Convey("List events", func() {
				eventList, err := fakeclientset.CoreV1().Events("fake-namespace").List(metav1.ListOptions{})

				convey.Convey("No error should be generated when listing the events", func() {
					convey.So(err, convey.ShouldBeEmpty)
				})

				convey.Convey("Only one event is generated", func() {

					convey.So(len(eventList.Items), convey.ShouldEqual, 1)

					convey.Convey("Event namespace must be fake-namespace", func() {
						convey.So(eventList.Items[0].Namespace, convey.ShouldEqual, "fake-namespace")
					})
				})
			})
		})
	})
}
