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

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
)

func TestHasher(t *testing.T) {
	convey.Convey("Given a value, hash it", t, func() {
		hash := Hasher("test")
		convey.So(hash, convey.ShouldNotBeEmpty)
	})
}

func TestDefaultConfigMapName(t *testing.T) {
	res := DefaultConfigMapName("sensor-controller")
	assert.Equal(t, "sensor-controller-configmap", res)
}

func TestServerResourceForGroupVersionKind(t *testing.T) {
	convey.Convey("Given a k8s client", t, func() {
		fakeClient := fake.NewSimpleClientset()
		fakeDisco := fakeClient.Discovery()
		gvk := schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Pod",
		}
		convey.Convey("Get a server resource for group, version and kind", func() {
			apiresource, err := ServerResourceForGroupVersionKind(fakeDisco, gvk)
			convey.Convey("Make sure error occurs and the resource is nil", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(apiresource, convey.ShouldBeNil)
			})
		})
	})
}
