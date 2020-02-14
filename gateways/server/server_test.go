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

package server

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc/metadata"
)

type FakeGRPCStream struct {
	SentData *gateways.Event
	Ctx      context.Context
}

func (f *FakeGRPCStream) Send(event *gateways.Event) error {
	f.SentData = event
	return nil
}

func (f *FakeGRPCStream) SetHeader(metadata.MD) error {
	return nil
}

func (f *FakeGRPCStream) SendHeader(metadata.MD) error {
	return nil
}

func (f *FakeGRPCStream) SetTrailer(metadata.MD) {
	return
}

func (f *FakeGRPCStream) Context() context.Context {
	return f.Ctx
}

func (f *FakeGRPCStream) SendMsg(m interface{}) error {
	return nil
}

func (f *FakeGRPCStream) RecvMsg(m interface{}) error {
	return nil
}

func TestHandleEventsFromEventSource(t *testing.T) {
	convey.Convey("Given a gateway server, handle events from an event source", t, func() {
		logger := common.NewArgoEventsLogger()
		channels := &Channels{
			Data: make(chan []byte),
			Stop: make(chan struct{}),
			Done: make(chan struct{}),
		}
		ctx, cancel := context.WithCancel(context.Background())

		convey.Convey("handle data", func() {
			es := &FakeGRPCStream{
				Ctx: ctx,
			}
			go HandleEventsFromEventSource("fake", es, channels, logger)
			channels.Data <- []byte("hello")
			time.Sleep(1 * time.Second)
			cancel()
			convey.So(string(es.SentData.Payload), convey.ShouldEqual, "hello")
		})
	})
}
