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

package aws_sqs

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListenEvents(t *testing.T) {
	convey.Convey("Given an event source, listen to events", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ps, convey.ShouldNotBeNil)

		ese := &SQSEventSourceExecutor{
			Clientset: fake.NewSimpleClientset(),
			Namespace: "fake",
			Log:       common.NewArgoEventsLogger(),
		}

		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{}, 1)
		errorCh2 := make(chan error)

		go func() {
			err := <-errorCh
			errorCh2 <- err
		}()

		ese.listenEvents(ps.(*sqsEventSource), &gateways.EventSource{
			Name: "fake",
			Data: es,
			Id:   "1234",
		}, dataCh, errorCh, doneCh)

		err = <-errorCh2
		convey.So(err, convey.ShouldNotBeNil)
	})

	convey.Convey("Given an event source without AWS credentials, listen to events", t, func() {
		ps, err := parseEventSource(esWithoutCreds)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ps, convey.ShouldNotBeNil)

		ese := &SQSEventSourceExecutor{
			Clientset: fake.NewSimpleClientset(),
			Namespace: "fake",
			Log:       common.NewArgoEventsLogger(),
		}

		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{}, 1)
		errorCh2 := make(chan error)

		go func() {
			err := <-errorCh
			errorCh2 <- err
		}()

		ese.listenEvents(ps.(*sqsEventSource), &gateways.EventSource{
			Name: "fake",
			Data: es,
			Id:   "1234",
		}, dataCh, errorCh, doneCh)

		err = <-errorCh2
		convey.So(err, convey.ShouldNotBeNil)
	})
}
