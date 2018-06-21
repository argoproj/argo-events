package artifact

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-events/job/shared"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	minio "github.com/minio/minio-go"
	"k8s.io/client-go/kubernetes"
)

const (
	EventType = "com.github.minio.bucket-notification"
	ISO8601   = "2006-01-02T15:04:05:07Z"
)

// s3 is a plugin for an S3 artifact signal
type s3 struct {
	kubeClient     kubernetes.Interface
	namespace      string
	signal         *v1alpha1.Signal
	streamSignaler shared.Signaler
}

// New creates a new S3 signal
func New(streamSignaler shared.Signaler, kubeClient kubernetes.Interface, nm string) shared.ArtifactSignaler {
	return &s3{
		streamSignaler: streamSignaler,
		kubeClient:     kubeClient,
		namespace:      nm,
	}
}

// Start the artifact signal
func (s *s3) Start(signal *v1alpha1.Signal) (<-chan shared.Event, error) {
	streamSignal, err := extractAndCreateStreamSignal(signal)
	if err != nil {
		return nil, err
	}
	streamEvents, err := s.streamSignaler.Start(streamSignal)
	if err != nil {
		return nil, err
	}
	events := make(chan shared.Event)
	go s.interceptFilterAndEnhanceEvents(events, streamEvents)
	return events, nil
}

// Stop the artifact signal
func (s *s3) Stop() error {
	return s.streamSignaler.Stop()
}

// method should be invoked as a separate go routine within the artifact Start method
// intercepts the receive-only msgs off the stream, filters them, and writes artifact events
// to the sendCh.
func (s *s3) interceptFilterAndEnhanceEvents(sendCh chan shared.Event, recvCh <-chan shared.Event) {
	defer close(sendCh)
	for streamEvent := range recvCh {
		// todo: apply general filtering on cloudEvents
		event := proto.Clone(&streamEvent).(*shared.Event)

		notification := &minio.NotificationInfo{}
		err := json.Unmarshal(streamEvent.Data, notification)
		if err != nil {
			// we ignore this - as this stream could be in use by another publisher of different notifications
			continue
		}
		if notification.Err != nil {
			event.Context.Extensions[shared.ContextExtensionErrorKey] = notification.Err.Error()
		}
		for _, record := range notification.Records {
			if ok := applyFilter(&record, s.signal.Artifact); !ok {
				// this record failed to pass the filter so we ignore it
				continue
			}
			port, _ := strconv.ParseInt(record.Source.Port, 10, 32)
			event.Context.EventType = EventType
			event.Context.EventTime = getProtoTimestamp(record.EventTime)
			event.Context.EventTypeVersion = record.EventVersion
			event.Context.Source = &shared.URI{
				Scheme: record.EventSource,
				User:   record.UserIdentity.PrincipalID,
				Host:   record.Source.Host,
				Port:   int32(port),
			}
			event.Context.SchemaURL = &shared.URI{
				Scheme: record.S3.SchemaVersion,
			}
			event.Context.EventID = record.S3.Object.ETag

			// read the actual s3 artifact to put into the event data
			b, err := s.Read(&s.signal.Artifact.ArtifactLocation, record.S3.Object.Key)
			if err != nil {
				event.Context.Extensions[shared.ContextExtensionErrorKey] = err.Error()
			}
			event.Data = b
			sendCh <- *event
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
		Name:        fmt.Sprintf("%s-artifact-stream", artifactSignal.Name),
		Stream:      &artifactSignal.Artifact.NotificationStream,
		Constraints: artifactSignal.Constraints,
	}, nil
}

// checks if the notification satisfies the signal
// 3 conditions must be met
// 1. notification bucket name must equal the S3 bucket
// 2. notification event name must equal the signal S3 event
// 3. notification object must pass the prefix and suffix string literals
func applyFilter(notification *minio.NotificationEvent, signal *v1alpha1.ArtifactSignal) bool {
	if signal.S3.Filter != nil {
		return notification.S3.Bucket.Name == signal.S3.Bucket &&
			notification.EventName == string(signal.S3.Event) &&
			strings.HasPrefix(notification.S3.Object.Key, signal.S3.Filter.Prefix) &&
			strings.HasSuffix(notification.S3.Object.Key, signal.S3.Filter.Suffix)
	}
	return notification.S3.Bucket.Name == signal.S3.Bucket &&
		notification.EventName == string(signal.S3.Event)
}

func getProtoTimestamp(tStr string) (timestamp *timestamp.Timestamp) {
	t, _ := time.Parse(ISO8601, tStr)
	timestamp, err := ptypes.TimestampProto(t)
	if err != nil {
		timestamp = ptypes.TimestampNow()
	}
	return
}

func (s *s3) Read(loc *v1alpha1.ArtifactLocation, key string) ([]byte, error) {
	creds, err := store.GetCredentials(s.kubeClient, s.namespace, loc)
	if err != nil {
		return nil, err
	}
	client, err := store.NewMinioClient(s.signal.Artifact.S3, *creds)
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
