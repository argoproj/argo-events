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

package gateways

import (
	"fmt"
	"github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
	"testing"
)

type MyMockedObject struct {
	mock.Mock
}

func (m *MyMockedObject) dispatchEventOverHttp(source string, eventPayload []byte) error {
	return nil
}

func (m *MyMockedObject) dispatchEventOverNats(source string, eventPayload []byte) error {
	return nil
}

func TestDispatchEvent(t *testing.T) {
	convey.Convey("Given an event, dispatch it to sensor", t, func() {
		gc := getGatewayConfig()
		event := &Event{
			Name:    "fake",
			Payload: []byte("fake"),
		}
		testObj := new(MyMockedObject)
		testObj.On("dispatchEventOverHttp", "fake", []byte("fake")).Return(nil)
		testObj.On("dispatchEventOverNats", "fake", []byte("fake")).Return(nil)

		err := gc.DispatchEvent(event)
		convey.So(err, convey.ShouldBeNil)

		gc.gw.Spec.EventProtocol.Type = common.NATS

		err = gc.DispatchEvent(event)
		convey.So(err, convey.ShouldBeNil)

		gc.gw.Spec.EventProtocol.Type = common.NATS

		err = gc.DispatchEvent(event)
		convey.So(err, convey.ShouldBeNil)

		gc.gw.Spec.EventProtocol.Type = common.EventProtocolType("fake")
		err = gc.DispatchEvent(event)
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestTransformEvent(t *testing.T) {
	convey.Convey("Given a gateway event, convert it into cloud event", t, func() {
		gc := getGatewayConfig()
		ce, err := gc.transformEvent(&Event{
			Name:    "fake",
			Payload: []byte("fake"),
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(ce, convey.ShouldNotBeNil)
		convey.So(ce.Context.Source.Host, convey.ShouldEqual, fmt.Sprintf("%s:%s", gc.gw.Name, "fake"))
	})
}
