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
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
)

func TestListenEvents(t *testing.T) {
	convey.Convey("Given a pubsub event source, listen to events", t, func() {
		ese := &EventSourceListener{
			Logger: common.NewArgoEventsLogger(),
		}

		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{}, 1)
		errCh2 := make(chan error)

		go func() {
			err := <-errorCh
			errCh2 <- err
		}()

		pubsubEventSource := &v1alpha1.PubSubEventSource{
			ProjectID: "1234",
			Topic:     "test",
		}

		body, err := yaml.Marshal(pubsubEventSource)
		convey.So(err, convey.ShouldBeNil)

		ese.listenEvents(context.Background(), &gateways.EventSource{
			Name:    "fake",
			Value:   body,
			Id:      "1234",
			Type:    string(v1alpha1.PubSubEvent),
			Version: "v0.10",
		}, dataCh, errorCh, doneCh)

		err = <-errCh2
		convey.So(err, convey.ShouldNotBeNil)
	})
}
