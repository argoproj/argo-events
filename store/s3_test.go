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

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"

	"github.com/smartystreets/goconvey/convey"
)

func TestNewMinioClient(t *testing.T) {
	convey.Convey("Given a configuration, get minio client", t, func() {
		client, err := NewMinioClient(&v1alpha1.S3Artifact{
			Endpoint: "fake",
			Region:   "us-east-1",
		}, Credentials{
			accessKey: "access",
			secretKey: "secret",
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(client, convey.ShouldNotBeNil)
	})
}

func TestNewS3Reader(t *testing.T) {
	convey.Convey("Given a minio client, get a reader", t, func() {
		reader, err := NewS3Reader(&v1alpha1.S3Artifact{
			Endpoint: "fake",
			Region:   "us-east-1",
		}, &Credentials{
			accessKey: "access",
			secretKey: "secret",
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(reader, convey.ShouldNotBeNil)
	})
}
