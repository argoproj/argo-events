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

package aws_sns

import (
	"context"
	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

var (
	configKey   = "testConfig"
	configId    = "1234"
	validConfig = `
hook:
 endpoint: "/test"
 port: "8080"
 url: "myurl/test"
topicArn: "test-arn"
region: "us-east-1"
accessKey:
    key: accesskey
    name: sns
secretKey:
    key: secretkey
    name: sns
`

	invalidConfig = `
endpoint: "/"
port: "8080"
`
)

func TestSNSEventSourceExecutor_ValidateEventSource(t *testing.T) {
	convey.Convey("Given a valid sns event source spec, parse it and make sure no error occurs", t, func() {
		ese := &SNSEventSourceExecutor{}
		valid, _ := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Name: configKey,
			Id:   configId,
			Data: validConfig,
		})
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(valid.IsValid, convey.ShouldBeTrue)
	})

	convey.Convey("Given an invalid sns event source spec, parse it and make sure error occurs", t, func() {
		ese := &SNSEventSourceExecutor{}
		valid, _ := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Data: invalidConfig,
			Id:   configId,
			Name: configKey,
		})
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(valid.IsValid, convey.ShouldBeFalse)
		convey.So(valid.Reason, convey.ShouldNotBeEmpty)
	})
}
