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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSetObjectMeta(t *testing.T) {
	convey.Convey("Given an object, set meta", t, func() {
		groupVersionKind := schema.GroupVersionKind{
			Group:   "grp",
			Version: "ver",
			Kind:    "kind",
		}
		owner := corev1.Pod{}
		pod := corev1.Pod{}
		ref := metav1.NewControllerRef(&owner, groupVersionKind)

		err := SetObjectMeta(&owner, &pod, groupVersionKind)
		convey.So(err, convey.ShouldBeEmpty)
		convey.So(pod.Labels["foo"], convey.ShouldEqual, "")
		convey.So(pod.Labels["id"], convey.ShouldEqual, "ID")
		convey.So(pod.Annotations, convey.ShouldContainKey, "hash")
		convey.So(pod.Name, convey.ShouldEqual, "")
		convey.So(pod.GenerateName, convey.ShouldEqual, "")
		convey.So(pod.OwnerReferences, convey.ShouldContain, *ref)
	})
}
