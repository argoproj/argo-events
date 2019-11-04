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

package resource

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

func TestValidateResourceEventSource(t *testing.T) {
	convey.Convey("Given a resource event source spec, parse it and make sure no error occurs", t, func() {
		listener := &EventListener{}
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", common.EventSourceDir, "resource.yaml"))
		convey.So(err, convey.ShouldBeNil)

		var eventSource *v1alpha1.EventSource
		err = yaml.Unmarshal(content, &eventSource)
		convey.So(err, convey.ShouldBeNil)
		convey.So(eventSource, convey.ShouldNotBeNil)

		err = v1alpha1.ValidateEventSource(eventSource)
		convey.So(err, convey.ShouldBeNil)

		for key, value := range eventSource.Spec.Resource {
			body, err := yaml.Marshal(value)
			convey.So(err, convey.ShouldBeNil)
			convey.So(err, convey.ShouldNotBeNil)

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
