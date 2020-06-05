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

package store

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNewResourceReader(t *testing.T) {
	convey.Convey("Given a resource, get new reader", t, func() {
		reader, err := NewResourceReader(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]string{
					"name": "mysecret",
				},
				"type": "Opaque",
				"data": map[string]string{
					"access": "c2VjcmV0",
					"secret": "c2VjcmV0",
				},
			},
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(reader, convey.ShouldNotBeNil)

		data, err := reader.Read()
		convey.So(err, convey.ShouldBeNil)
		convey.So(data, convey.ShouldNotBeNil)
	})
}
