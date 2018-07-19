package artifact

import (
	"reflect"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
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

/*
func TestSignal(t *testing.T) {
	natsEmbeddedServerOpts := server.Options{
		Host:           "localhost",
		Port:           4223,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 256,
	}
	s3 := New(nats.New(), fake.NewSimpleClientset(), "default")

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
			NotificationStream: v1alpha1.Stream{
				Type:       "NATS",
				URL:        "nats://" + natsEmbeddedServerOpts.Host + ":" + strconv.Itoa(natsEmbeddedServerOpts.Port),
				Attributes: map[string]string{"subject": "test"},
			},
		},
	}

	// start the signal
	_, err := s3.Start(&signal)
	if err == nil {
		t.Errorf("expected: failed to connect to nats cluster\nfound: %s", err)
	}

	// run an embedded gnats server
	testServer := test.RunServer(&natsEmbeddedServerOpts)
	defer testServer.Shutdown()
	events, err := s3.Start(&signal)
	if err != nil {
		t.Error(err)
	}

	// send a misformed message - should not get anything on channel
	conn, err := natsio.Connect(signal.Artifact.NotificationStream.URL)
	if err != nil {
		t.Fatalf("failed to connect to embedded nats server. cause: %s", err)
	}
	defer conn.Close()
	err = conn.Publish("test", []byte("hello, world"))
	if err != nil {
		t.Fatalf("failed to publish misformed msg. cause: %s", err)
	}

	// send a minio bucket notification with error
	err = conn.Publish("test", []byte(notificationInfoWithErr))
	if err != nil {
		t.Fatalf("failed to publish notification msg. cause: %s", err)
	}
	event, ok := <-events
	if !ok {
		t.Errorf("expected an event but found none")
	}
	// verify the error extension
	errMsg := event.Context.Extensions[shared.ContextExtensionErrorKey]
	if errMsg != "this is an error" {
		t.Errorf("event context error:\nexpected: %s\nactual: %s", "this is an error", errMsg)
	}

	// send a valid minio bucket notification
	err = conn.Publish("test", []byte(notificationInfo))
	if err != nil {
		t.Fatalf("failed to publish notification msg. cause: %s", err)
	}
	event, ok = <-events
	if !ok {
		t.Errorf("expected an event but found none")
	}
	// verify the event data
	if event.Context.EventType != EventType {
		t.Errorf("event context EventType:\nexpected: %s\nactual: %s", EventType, event.Context.EventID)
	}

	err = s3.Stop()
	if err != nil {
		t.Errorf("failed to stop signal. cause: %s", err)
	}

	// ensure events channel is closed
	if _, ok := <-events; ok {
		t.Errorf("expected read-only events channel to be closed after signal stop")
	}
}
*/

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

var notificationInfoWithErr = `
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
	"Err": "this is an error",
	"level": "info",
	"msg": "",
	"time": "2017-07-07T11:46:37-07:00"
}
`

/*
func TestGetProtoTimestamp(t *testing.T) {
	tStr := "1970-01-01T00:00:00.00Z"
	//"2006-01-02T15:04:05:07Z"
	timestamp1 := getProtoTimestamp(tStr)

	t1, err := ptypes.Timestamp(timestamp1)
	if err != nil {
		t.Error(err)
	}
	expected := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !t1.Equal(expected) {
		t.Errorf("times are not equal\nexpected: %s\nactual: %s", expected, t1)
	}
}
*/
