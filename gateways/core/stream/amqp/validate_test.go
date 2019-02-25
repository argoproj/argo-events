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

package amqp

import (
	"context"
	"testing"

	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
)

var (
	configKey   = "testConfig"
	configId    = "1234"
	configValue = `
url: amqp://amqp.argo-events:5672
exchangeName: fooExchangeName
exchangeType: fanout
routingKey: fooRoutingKey
`
	invalidConfig = `
url: amqp://amqp.argo-events:5672
exchangeName: fooExchangeName
exchangeType: fanout
`
)

func TestValidateAMQPEventSource(t *testing.T) {
	convey.Convey("Given a valid amqp event source spec, parse it and make sure no error occurs", t, func() {
		ese := &AMQPEventSourceExecutor{}
		valid, _ := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Name: configKey,
			Id:   configId,
			Data: configValue,
		})
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(valid.IsValid, convey.ShouldBeTrue)
	})

	convey.Convey("Given an invalid amqp event source spec, parse it and make sure error occurs", t, func() {
		ese := &AMQPEventSourceExecutor{}
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
