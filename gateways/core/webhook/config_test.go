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

package webhook

import (
	"testing"

	"github.com/argoproj/argo-events/gateways/common"
	"github.com/smartystreets/goconvey/convey"
)

var es = `
endpoint: "/bar"
port: "10000"
method: "POST"
`

func TestParseConfig(t *testing.T) {
	convey.Convey("Given a webhook event source, parse it", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ps, convey.ShouldNotBeNil)
		_, ok := ps.(*common.Webhook)
		convey.So(ok, convey.ShouldEqual, true)
	})
}
