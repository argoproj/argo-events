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
	"fmt"
	"github.com/smartystreets/goconvey/convey"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
)

type fakeHttpWriter struct {
	header  int
	payload []byte
}

func (f *fakeHttpWriter) Header() http.Header {
	return http.Header{}
}

func (f *fakeHttpWriter) Write(body []byte) (int, error) {
	f.payload = body
	return len(body), nil
}

func (f *fakeHttpWriter) WriteHeader(status int) {
	f.header = status
}

func TestGetObjectHash(t *testing.T) {
	convey.Convey("Given a value, hash it", t, func() {
		hash, err := GetObjectHash(&corev1.Pod{})
		convey.So(hash, convey.ShouldNotBeEmpty)
		convey.So(err, convey.ShouldBeEmpty)
	})
}

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

func TestDefaultServiceName(t *testing.T) {
	convey.Convey("Given a service, get the default name", t, func() {
		convey.So(DefaultServiceName("default"), convey.ShouldEqual, fmt.Sprintf("%s-svc", "default"))
	})
}

func TestDefaultNatsQueueName(t *testing.T) {
	convey.Convey("Given a nats queue, get the default name", t, func() {
		convey.So(DefaultNatsQueueName("sensor", "default"), convey.ShouldEqual, "sensor-default-queue")
	})
}

func TestHTTPMethods(t *testing.T) {
	convey.Convey("Given a http write", t, func() {
		convey.Convey("Write a success response", func() {
			f := &fakeHttpWriter{}
			SendSuccessResponse(f, "hello")
			convey.So(string(f.payload), convey.ShouldEqual, "hello")
			convey.So(f.header, convey.ShouldEqual, http.StatusOK)
		})

		convey.Convey("Write a failure response", func() {
			f := &fakeHttpWriter{}
			SendErrorResponse(f, "failure")
			convey.So(string(f.payload), convey.ShouldEqual, "failure")
			convey.So(f.header, convey.ShouldEqual, http.StatusBadRequest)
		})
	})
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
