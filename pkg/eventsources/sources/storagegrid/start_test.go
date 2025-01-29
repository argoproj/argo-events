/*
Copyright 2018 The Argoproj Authors.

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
	"io"
	"net/http"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
)

var (
	notification = `
{
  "Action": "Publish",
  "Message": {
    "Records": [
      {
        "eventName": "ObjectCreated:Put",
        "storageGridEventSource": "sgws:s3",
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
	router = &Router{
		route: webhook.GetFakeRoute(),
	}
)

func TestRouteActiveHandler(t *testing.T) {
	convey.Convey("Given a route configuration", t, func() {
		storageGridEventSource := &v1alpha1.StorageGridEventSource{
			Webhook: &v1alpha1.WebhookContext{
				Endpoint: "/",
				URL:      "testurl",
				Port:     "8080",
			},
			Events: []string{
				"ObjectCreated:Put",
			},
			Filter: &v1alpha1.StorageGridFilter{
				Prefix: "hello-",
				Suffix: ".txt",
			},
		}

		writer := &webhook.FakeHttpWriter{}

		convey.Convey("Inactive route should return error", func() {
			pbytes, err := yaml.Marshal(storageGridEventSource)
			convey.So(err, convey.ShouldBeNil)
			router.HandleRoute(writer, &http.Request{
				Body: io.NopCloser(bytes.NewReader(pbytes)),
			})
			convey.So(writer.HeaderStatus, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("Active route should return success", func() {
			router.route.Active = true
			router.storageGridEventSource = storageGridEventSource
			dataCh := make(chan *webhook.Dispatch)
			go func() {
				resp := <-router.route.DispatchChan
				dataCh <- resp
			}()

			router.HandleRoute(writer, &http.Request{
				Body: io.NopCloser(bytes.NewReader([]byte(notification))),
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

func TestFilterName(t *testing.T) {
	convey.Convey("Given a storage grid event, test whether the object key passes the filter", t, func() {
		storageGridEventSource := &v1alpha1.StorageGridEventSource{
			Webhook: &v1alpha1.WebhookContext{
				Endpoint: "/",
				URL:      "testurl",
				Port:     "8080",
			},
			Events: []string{
				"ObjectCreated:Put",
			},
			Filter: &v1alpha1.StorageGridFilter{
				Prefix: "hello-",
				Suffix: ".txt",
			},
		}
		var gridNotification *events.StorageGridNotification
		err := json.Unmarshal([]byte(notification), &gridNotification)
		convey.So(err, convey.ShouldBeNil)
		convey.So(gridNotification, convey.ShouldNotBeNil)

		ok := filterName(gridNotification, storageGridEventSource)
		convey.So(ok, convey.ShouldEqual, true)
	})
}
