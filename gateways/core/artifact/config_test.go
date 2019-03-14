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

package artifact

import (
	"testing"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/smartystreets/goconvey/convey"
)

var es = `
bucket:
    name: input
endpoint: minio-service.argo-events:9000
event: s3:ObjectCreated:Put
filter:
    prefix: ""
    suffix: ""
insecure: true
accessKey:
    key: accesskey
    name: artifacts-minio
secretKey:
    key: secretkey
    name: artifacts-minio
`

func TestParseConfig(t *testing.T) {
	convey.Convey("Given a artifact event source, parse it", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		convey.So(ps, convey.ShouldNotBeNil)
		_, ok := ps.(*apicommon.S3Artifact)
		convey.So(ok, convey.ShouldEqual, true)
	})
}
