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
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
)

func TestGcpPubSubEventSourceExecutor_ValidateEventSource(t *testing.T) {
	convey.Convey("Given a valid gcp pub-sub event source spec, parse it and make sure no error occurs", t, func() {
		listener := &EventSourceListener{}
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", common.EventSourceDir, "gcp-pubsub.yaml"))
		convey.So(err, convey.ShouldBeNil)

		var eventSource *v1alpha1.EventSource
		err = yaml.Unmarshal(content, &eventSource)
		convey.So(err, convey.ShouldBeNil)
		convey.So(eventSource, convey.ShouldNotBeNil)

		for key, value := range eventSource.Spec.PubSub {
			body, err := yaml.Marshal(value)
			convey.So(err, convey.ShouldBeNil)
			convey.So(body, convey.ShouldNotBeNil)

			valid, _ := listener.ValidateEventSource(context.Background(), &gateways.EventSource{
				Name:    key,
				Id:      common.Hasher(key),
				Value:   body,
				Version: eventSource.Spec.Version,
				Type:    string(eventSource.Spec.Type),
			})
			convey.So(valid, convey.ShouldNotBeNil)
			convey.So(valid.IsValid, convey.ShouldBeTrue)
		}
	})
}
