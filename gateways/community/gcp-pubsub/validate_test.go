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

package pubsub

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
projectID: "1234"
topic: "test"
`
	invalidConfig = `
projectID: "1234"
`
)

func TestGcpPubSubEventSourceExecutor_ValidateEventSource(t *testing.T) {
	convey.Convey("Given a valid gcp pub-sub event source spec, parse it and make sure no error occurs", t, func() {
		ese := &GcpPubSubEventSourceExecutor{}

		valid, _ := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Name: configKey,
			Id:   configId,
			Data: validConfig,
		})
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(valid.IsValid, convey.ShouldBeTrue)
	})

	convey.Convey("Given an invalid gcp pub-sub event source spec, parse it and make sure error occurs", t, func() {
		ese := &GcpPubSubEventSourceExecutor{}
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
