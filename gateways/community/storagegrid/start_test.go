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

package storagegrid

import (
	"bytes"
	"encoding/json"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net/http"
	"testing"
)

var (
	notification = `
{
  "Action": "Publish",
  "Message": {
    "Records": [
      {
        "eventName": "ObjectCreated:Put",
        "eventSource": "sgws:s3",
        "eventTime": "2019-02-27T21:15:09Z",
        "eventVersion": "2.0",
        "requestParameters": {
          "sourceIPAddress": "1.1.1.1"
        },
        "responseElements": {
          "x-amz-request-id": "12345678"
        },
        "s3": {
          "bucket": {
            "arn": "urn:sgfs:s3:::my_bucket",
            "name": "my_bucket",
            "ownerIdentity": {
              "principalId": "55555555555555555"
            }
          },
          "configurationId": "Object-Event",
          "object": {
            "eTag": "4444444444444444",
            "key": "hello-world.txt",
            "sequencer": "AAAAAA",
            "size": 6
          },
          "s3SchemaVersion": "1.0"
        },
        "userIdentity": {
          "principalId": "1111111111111111"
        }
      }
    ]
  },
  "TopicArn": "urn:h:sns:us-east::my_topic_1",
  "Version": "2010-03-31"
}
`
	rc = &RouteConfig{
		route: gwcommon.GetFakeRoute(),
	}
)

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		helper.ActiveEndpoints[rc.route.Webhook.Endpoint] = &gwcommon.Endpoint{
			DataCh: make(chan []byte),
		}

		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		writer := &gwcommon.FakeHttpWriter{}

		convey.Convey("Inactive route should return error", func() {
			pbytes, err := yaml.Marshal(ps.(*storageGridEventSource))
			convey.So(err, convey.ShouldBeNil)
			rc.RouteHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(pbytes)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("Active route should return success", func() {
			helper.ActiveEndpoints[rc.route.Webhook.Endpoint].Active = true
			rc.sges = ps.(*storageGridEventSource)
			dataCh := make(chan []byte)
			go func() {
				resp := <-helper.ActiveEndpoints[rc.route.Webhook.Endpoint].DataCh
				dataCh <- resp
			}()

			rc.RouteHandler(writer, &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(notification))),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusOK)
		})
	})
}

func TestGenerateUUID(t *testing.T) {
	convey.Convey("Make sure generated UUIDs are unique", t, func() {
		u1 := generateUUID()
		u2 := generateUUID()
		convey.So(u1.String(), convey.ShouldNotEqual, u2.String())
	})
}

func TestFilterEvent(t *testing.T) {
	convey.Convey("Given a storage grid event, test whether it passes the filter", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		var sg *storageGridNotification
		err = json.Unmarshal([]byte(notification), &sg)
		convey.So(err, convey.ShouldBeNil)
		convey.So(sg, convey.ShouldNotBeNil)

		ok := filterEvent(sg, ps.(*storageGridEventSource))
		convey.So(ok, convey.ShouldEqual, true)
	})
}

func TestFilterName(t *testing.T) {
	convey.Convey("Given a storage grid event, test whether the object key passes the filter", t, func() {
		ps, err := parseEventSource(es)
		convey.So(err, convey.ShouldBeNil)
		var sg *storageGridNotification
		err = json.Unmarshal([]byte(notification), &sg)
		convey.So(err, convey.ShouldBeNil)
		convey.So(sg, convey.ShouldNotBeNil)

		ok := filterName(sg, ps.(*storageGridEventSource))
		convey.So(ok, convey.ShouldEqual, true)
	})
}
