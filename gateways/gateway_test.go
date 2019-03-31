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
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/smartystreets/goconvey/convey"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

type FakeGRPCStream struct {
	SentData *Event
	Ctx      context.Context
}

func (f *FakeGRPCStream) Send(event *Event) error {
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
		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{})
		logger := common.NewArgoEventsLogger()

		ctx, cancel := context.WithCancel(context.Background())

		convey.Convey("handle data", func() {
			es := &FakeGRPCStream{
				Ctx: ctx,
			}
			go HandleEventsFromEventSource("fake", es, dataCh, errorCh, doneCh, logger)
			dataCh <- []byte("hello")
			time.Sleep(1 * time.Second)
			cancel()
			convey.So(string(es.SentData.Payload), convey.ShouldEqual, "hello")
		})

		convey.Convey("handle error", func() {
			es := &FakeGRPCStream{
				Ctx: ctx,
			}
			errorCh2 := make(chan error)
			go func() {
				err := HandleEventsFromEventSource("fake", es, dataCh, errorCh, doneCh, logger)
				errorCh2 <- err
			}()
			errorCh <- fmt.Errorf("fake error")
			err := <-errorCh2
			convey.So(err.Error(), convey.ShouldEqual, "fake error")
		})
	})
}
