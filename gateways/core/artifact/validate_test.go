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

package artifact

import (
	"context"
	"github.com/smartystreets/goconvey/convey"
	"testing"

	"github.com/argoproj/argo-events/gateways"
)

var (
	configKey   = "testConfig"
	configValue = `
s3EventConfig:
    bucket: input
    endpoint: minio-service.argo-events:9000
    event: s3:ObjectCreated:Put
    filter:
    prefix: ""
    suffix: ""
insecure: true
accessKey:
    key: accesskey
    name: artifacts-minio
secretKey:
    key: secretkey
    name: artifacts-minio
`
)

func TestValidateS3EventSource(t *testing.T) {
	convey.Convey("Given a valid S3 artifact spec, parse the spec and make sure no error occurs", t, func() {
		ese := &S3EventSourceExecutor{}
		name := "artifact"
		id := "1234"
		valid, err := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Name: &name,
			Data: &configValue,
			Id: &id,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(*valid.IsValid, convey.ShouldBeTrue)
	})

	convey.Convey("Given an invalid S3 artifact spec", t, func() {
		ese := &S3EventSourceExecutor{}
		name := "artifact"
		id := "1234"
		invalidS3Artifact := `
s3EventConfig:
    bucket: input
    endpoint: minio-service.argo-events:9000
    event: s3:ObjectCreated:Put
    filter:
        prefix: ""
        suffix: ""
insecure: true
`
		valid, err := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Id: &id,
			Name: &name,
			Data: &invalidS3Artifact,
		})
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(*valid.IsValid, convey.ShouldBeFalse)
		convey.So(*valid.Reason, convey.ShouldNotBeEmpty)
	})
}
