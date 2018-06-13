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
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	minio "github.com/minio/minio-go"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/gnatsd/test"
	natsio "github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/blackrock/axis/job"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
)

func TestSignal(t *testing.T) {
	natsEmbeddedServerOpts := server.Options{
		Host:           "localhost",
		Port:           4223,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 256,
	}
	es := job.New(nil, nil, zap.NewNop())
	Artifact(es)
	artifactFactory, ok := es.GetCoreFactory(v1alpha1.SignalTypeArtifact)
	assert.True(t, ok, "artifact factory is not found")

	abstractSignal := job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Name: "artifact-nats-test",
			Artifact: &v1alpha1.ArtifactSignal{
				ArtifactLocation: v1alpha1.ArtifactLocation{
					S3: &v1alpha1.S3Artifact{
						S3Bucket: v1alpha1.S3Bucket{
							Bucket: "images",
						},
						Event: minio.ObjectCreatedPut,
						Filter: &v1alpha1.S3Filter{
							Prefix: "my",
							Suffix: ".jpg",
						},
					},
				},
				NotificationStream: v1alpha1.Stream{
					Type:       "NATS",
					URL:        "nats://" + natsEmbeddedServerOpts.Host + ":" + strconv.Itoa(natsEmbeddedServerOpts.Port),
					Attributes: map[string]string{"subject": "test"},
				},
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}

	signal, err := artifactFactory.Create(abstractSignal)
	assert.Nil(t, err)

	testCh := make(chan job.Event)

	// failure to start because of connecting to NATS
	err = signal.Start(testCh)
	assert.NotNil(t, err)

	// run an embedded gnats server
	testServer := test.RunServer(&natsEmbeddedServerOpts)
	defer testServer.Shutdown()
	err = signal.Start(testCh)
	assert.Nil(t, err)

	// send a misformed message
	conn, err := natsio.Connect(abstractSignal.Signal.Artifact.NotificationStream.URL)
	defer conn.Close()
	assert.Nil(t, err)
	err = conn.Publish("test", []byte("hello, world"))
	assert.Nil(t, err)

	event := <-testCh
	assert.NotNil(t, event.GetError())

	// send a minio bucket notification with error
	notification := minio.NotificationInfo{
		Records: make([]minio.NotificationEvent, 0),
		Err:     fmt.Errorf("this is a test error"),
	}
	b, err := json.Marshal(notification)
	assert.Nil(t, err)
	err = conn.Publish("test", b)
	assert.Nil(t, err)

	event = <-testCh
	assert.NotNil(t, event.GetError())

	// send a valid minio bucket notification
	err = conn.Publish("test", []byte(notificationInfo))
	assert.Nil(t, err)

	event = <-testCh
	assert.Equal(t, "", event.GetID())
	assert.NotNil(t, event.GetBody())
	assert.Equal(t, "images/myphoto.jpg", event.GetSource())
	assert.True(t, time.Now().After(event.GetTimestamp()))
	assert.Equal(t, signal, event.GetSignal())

	err = signal.Stop()
	assert.Nil(t, err)

	abstractSignal.Artifact.S3.Filter = nil
	signal, err = artifactFactory.Create(abstractSignal)
	assert.Nil(t, err)
	signal.Start(testCh)
	// now filter out a bucket notification
	err = conn.Publish("test", []byte(filteredNotification))
	assert.Nil(t, err)

	err = signal.Stop()
	assert.Nil(t, err)
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

var filteredNotification = `
{
	"EventType": "s3:ObjectCreated:Delete",
	"Key": "images/myphoto.jpg",
	"Records": [
	  {
		"eventVersion": "2.0",
		"eventSource": "minio:s3",
		"awsRegion": "",
		"eventTime": "2017-07-07T18:46:37Z",
		"eventName": "s3:ObjectCreated:Delete",
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
