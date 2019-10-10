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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestListenEvents(t *testing.T) {
	convey.Convey("Given a pubsub event source, listen to events", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ps, convey.ShouldNotBeNil)
		psc := ps.(*pubSubEventSource)

		ese := &GcpPubSubEventSourceExecutor{
			Log: common.NewArgoEventsLogger(),
		}

		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{}, 1)
		errCh2 := make(chan error)

		go func() {
			err := <-errorCh
			errCh2 <- err
		}()

		ese.listenEvents(context.Background(), psc, &gateways.EventSource{
			Name: "fake",
			Data: es,
			Id:   "1234",
		}, dataCh, errorCh, doneCh)

		err = <-errCh2
		convey.So(err, convey.ShouldNotBeNil)
	})
}
