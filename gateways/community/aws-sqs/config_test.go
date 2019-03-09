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

	"github.com/smartystreets/goconvey/convey"
)

var es = `
region: "us-east-1"
accessKey:
    key: accesskey
    name: sns
secretKey:
    key: secretkey
    name: sns
queue: "test-queue"
waitTimeSeconds: 10
`

func TestParseConfig(t *testing.T) {
	convey.Convey("Given a aws-sqs event source, parse it", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ps, convey.ShouldNotBeNil)
		_, ok := ps.(*sqs)
		convey.So(ok, convey.ShouldEqual, true)
	})
}
