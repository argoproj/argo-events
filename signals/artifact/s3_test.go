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
	"reflect"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk/fake"
	minio "github.com/minio/minio-go"
)

func TestExtractAndCreateStreamSignal(t *testing.T) {
	empty := v1alpha1.Signal{
		Name: "empty",
	}
	notificationStream := v1alpha1.Stream{
		Type:       "NATS",
		URL:        "nats://localhost:4222",
		Attributes: map[string]string{"subject": "test"},
	}
	signal := v1alpha1.Signal{
		Name: "s3-test",
		Artifact: &v1alpha1.ArtifactSignal{
			ArtifactLocation: v1alpha1.ArtifactLocation{
				S3: &v1alpha1.S3Artifact{
					S3Bucket: v1alpha1.S3Bucket{
						Bucket: "bucket",
					},
				},
			},
			Target: notificationStream,
		},
	}
	_, err := extractAndCreateStreamSignal(&empty)
	if err == nil {
		t.Errorf("expected error for undefined artifact signal but found none")
	}
	streamSignal, err := extractAndCreateStreamSignal(&signal)
	if err != nil {
		t.Error(err)
	}

	if streamSignal.Stream == nil {
		t.Fatalf("signal stream is undefined")
	}
	if !reflect.DeepEqual(notificationStream, *streamSignal.Stream) {
		t.Errorf("stream defs are not equal\nexpected: %v\nactual: %v", notificationStream, streamSignal.Stream)
	}
}

func TestSignal(t *testing.T) {
	fakeClient := fake.NewClient()
	s3 := New(fakeClient)

	signal := v1alpha1.Signal{
		Name: "s3-test",
		Artifact: &v1alpha1.ArtifactSignal{
			ArtifactLocation: v1alpha1.ArtifactLocation{
				S3: &v1alpha1.S3Artifact{
					S3Bucket: v1alpha1.S3Bucket{
						Bucket: "images",
					},
					Key:   "myObj.txt",
					Event: minio.ObjectCreatedPut,
				},
			},
			Target: v1alpha1.Stream{
				Type: "TEST",
				URL:  "http://unit-tests-are-awesome.com",
			},
		},
	}

	done := make(chan struct{})
	events, err := s3.Listen(&signal, done)
	if err != nil {
		t.Error(err)
	}

	// send a proper notification
	notificationEvent := v1alpha1.Event{
		Context: v1alpha1.EventContext{},
		Data:    []byte(notificationInfo),
	}
	fakeClient.Generate(&notificationEvent)
	event, ok := <-events
	if !ok {
		t.Errorf("expected an event but found none")
	}
	// verify the event data
	if event.Context.EventType != EventType {
		t.Errorf("event context EventType:\nexpected: %s\nactual: %s", EventType, event.Context.EventID)
	}

	close(done)
	// ensure events channel is closed
	if _, ok := <-events; ok {
		t.Errorf("expected read-only events channel to be closed after signal stop")
	}
}

var notificationInfo = `
{
  "EventType": "s3:ObjectCreated:Put",
  "Key": "images/myphoto.jpg",
  "Records": [
    {
    "eventVersion": "2.0",
    "eventSource": "minio:s3",
    "awsRegion": "",
    "eventTime": "2017-07-07T18:46:37Z",
    "eventName": "s3:ObjectCreated:Put",
    "userIdentity": {
      "principalId": "minio"
    },
    "requestParameters": {
      "sourceIPAddress": "192.168.1.80:55328"
    },
    "responseElements": {
      "x-amz-request-id": "14CF20BD1EFD5B93",
      "x-minio-origin-endpoint": "http://127.0.0.1:9000"
    },
    "s3": {
      "s3SchemaVersion": "1.0",
      "configurationId": "Config",
      "bucket": {
      "name": "images",
      "ownerIdentity": {
        "principalId": "minio"
      },
      "arn": "arn:aws:s3:::images"
      },
      "object": {
      "key": "myphoto.jpg",
      "size": 248682,
      "eTag": "f1671feacb8bbf7b0397c6e9364e8c92",
      "contentType": "image/jpeg",
      "userDefined": {
        "content-type": "image/jpeg"
      },
      "versionId": "1",
      "sequencer": "14CF20BD1EFD5B93"
      }
    },
    "source": {
      "host": "192.168.1.80",
      "port": "55328",
      "userAgent": "Minio (linux; amd64) minio-go/2.0.4 mc/DEVELOPMENT.GOGET"
    }
    }
  ],
  "level": "info",
  "msg": "",
  "time": "2017-07-07T11:46:37-07:00"
}
`
