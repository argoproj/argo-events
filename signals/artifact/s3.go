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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	"github.com/argoproj/argo-events/store"
	"github.com/golang/protobuf/proto"
	minio "github.com/minio/minio-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	EventType = "com.github.minio.bucket-notification"
	ISO8601   = "2006-01-02T15:04:05:07Z"
)

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
// Listen() methods CAN retrieve fields from the s3 struct.
type s3 struct {
	kubeClient   kubernetes.Interface
	namespace    string
	streamClient sdk.SignalClient
}

// New creates a new S3 signal
func New(client sdk.SignalClient, kubeClient kubernetes.Interface, nm string) sdk.ArtifactListener {
	return &s3{
		streamClient: client,
		kubeClient:   kubeClient,
		namespace:    nm,
	}
}

func (s *s3) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	ctx := context.TODO()
	streamSignal, err := extractAndCreateStreamSignal(signal)
	if err != nil {
		return nil, err
	}
	stream, err := s.streamClient.Listen(ctx, streamSignal)
	if err != nil {
		return nil, err
	}

	// read from stream & write to streamEvents
	streamEvents := make(chan *v1alpha1.Event)
	go func() {
		defer close(streamEvents)
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Panicf("signal target stream received error: %s", err)
			}
			streamEvents <- in.Event
		}
	}()

	events := make(chan *v1alpha1.Event)

	// start stream receiver
	go s.interceptFilterAndEnhanceEvents(signal, events, streamEvents)

	// wait for stop signal
	go func() {
		<-done
		close(events)
		// TODO: should we cancel or gracefully shutdown and first send Terminate msg
		err := stream.Send(sdk.Terminate)
		if err != nil {
			log.Panicf("failed to terminate stream signal: %s", err)
		}
		err = stream.Close()
		if err != nil {
			log.Panicf("failed to close stream: %s", err)
		}
	}()
	return events, nil
}

// method should be invoked as a separate go routine within the artifact Start method
// intercepts the receive-only msgs off the stream, filters them, and writes artifact events
// to the sendCh.
func (s *s3) interceptFilterAndEnhanceEvents(sig *v1alpha1.Signal, sendCh chan *v1alpha1.Event, recvCh <-chan *v1alpha1.Event) {
	defer close(sendCh)
	for streamEvent := range recvCh {
		// todo: apply general filtering on cloudEvents
		event := proto.Clone(streamEvent).(*v1alpha1.Event)

		notification := &minio.NotificationInfo{}
		err := json.Unmarshal(streamEvent.Data, notification)
		if err != nil {
			// we ignore this - as this stream could be in use by another publisher of different notifications
			continue
		}
		if notification.Err != nil {
			event.Context.Extensions[sdk.ContextExtensionErrorKey] = notification.Err.Error()
		}
		for _, record := range notification.Records {
			if ok := applyFilter(&record, sig.Artifact.ArtifactLocation); !ok {
				// this record failed to pass the filter so we ignore it
				continue
			}
			port, _ := strconv.ParseInt(record.Source.Port, 10, 32)
			event.Context.EventType = EventType
			event.Context.EventTime = getMetaTimestamp(record.EventTime)
			event.Context.EventTypeVersion = record.EventVersion
			event.Context.Source = &v1alpha1.URI{
				Scheme: record.EventSource,
				User:   record.UserIdentity.PrincipalID,
				Host:   record.Source.Host,
				Port:   int32(port),
			}
			event.Context.SchemaURL = &v1alpha1.URI{
				Scheme: record.S3.SchemaVersion,
			}
			event.Context.EventID = record.S3.Object.ETag

			// read the actual s3 artifact to put into the event data
			b, err := s.Read(&sig.Artifact.ArtifactLocation, record.S3.Object.Key)
			if err != nil {
				event.Context.Extensions[sdk.ContextExtensionErrorKey] = err.Error()
			}
			event.Data = b
			sendCh <- event
		}
	}
}

// utility method to extract the stream definition from within the artifact signal definition.
// used to reconfigure the artifact signal to create a first class stream signal
func extractAndCreateStreamSignal(artifactSignal *v1alpha1.Signal) (*v1alpha1.Signal, error) {
	if artifactSignal.Artifact == nil {
		return nil, errors.New("undefined artifact signal")
	}
	return &v1alpha1.Signal{
		Name:     fmt.Sprintf("%s-artifact-stream", artifactSignal.Name),
		Deadline: artifactSignal.Deadline,
		Stream:   &artifactSignal.Artifact.Target,
		Filters:  artifactSignal.Filters,
	}, nil
}

// checks if the notification satisfies the signal
// 3 conditions must be met
// 1. notification bucket name must equal the S3 bucket
// 2. notification event name must equal the signal S3 event
// 3. notification object must pass the prefix and suffix string literals
func applyFilter(notification *minio.NotificationEvent, loc v1alpha1.ArtifactLocation) bool {
	if loc.S3.Filter != nil {
		return notification.S3.Bucket.Name == loc.S3.Bucket &&
			notification.EventName == string(loc.S3.Event) &&
			strings.HasPrefix(notification.S3.Object.Key, loc.S3.Filter.Prefix) &&
			strings.HasSuffix(notification.S3.Object.Key, loc.S3.Filter.Suffix)
	}
	return notification.S3.Bucket.Name == loc.S3.Bucket &&
		notification.EventName == string(loc.S3.Event)
}

func getMetaTimestamp(tStr string) metav1.Time {
	t, err := time.Parse(ISO8601, tStr)
	if err != nil {
		return metav1.Time{Time: time.Now().UTC()}
	}
	return metav1.Time{Time: t}
}

func (s *s3) Read(loc *v1alpha1.ArtifactLocation, key string) ([]byte, error) {
	creds, err := store.GetCredentials(s.kubeClient, s.namespace, loc)
	if err != nil {
		return nil, err
	}
	client, err := store.NewMinioClient(loc.S3, *creds)
	if err != nil {
		return nil, err
	}

	obj, err := client.GetObject(loc.S3.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	b, err := ioutil.ReadAll(obj)
	if err != nil {
		return nil, err
	}
	return b, nil
}
