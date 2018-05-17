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
	"strings"

	"go.uber.org/zap"

	"github.com/minio/minio-go"

	"github.com/blackrock/axis/job"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
)

type artifact struct {
	job.AbstractSignal
	streamSignal job.Signal
	streamEvents chan job.Event
}

func (a *artifact) Start(events chan job.Event) error {
	// we should not be in the business of creating/deleting bucket notifications as we can only delete by unique ARNs
	// this means we would disable notifications for other sensors inside the context of this sensor/job
	// this job should be delegated to a process that has knowledge of all the consumers of bucket notifications
	a.streamEvents = make(chan job.Event)
	err := a.streamSignal.Start(a.streamEvents)
	if err != nil {
		return err
	}
	go a.interceptFilterAndEnhanceStreamEvents(events, a.streamEvents)
	return nil
}

func (a *artifact) interceptFilterAndEnhanceStreamEvents(sendCh chan job.Event, receiveCh <-chan job.Event) {
	for streamEvent := range receiveCh {
		event := &event{
			artifact: a,
		}

		notification := &minio.NotificationInfo{}
		err := json.Unmarshal(streamEvent.GetBody(), notification)
		if err != nil {
			event.SetError(err)
			sendCh <- event
			continue
		}
		if notification.Err != nil {
			event.SetError(err)
			sendCh <- event
			continue
		}
		for _, record := range notification.Records {
			event.s3Notification = &record

			// filter the event via the bucket, event, and key
			if checkArtifactSignal(record, *a.AbstractSignal.Artifact) {
				// ignore the constraints set for this guy as it's for a different signal
				// perform constraint checks against the artifact signal
				err := a.CheckConstraints(event.GetTimestamp())
				if err != nil {
					event.SetError(err)
				}
				a.Log.Debug("sending artifact event", zap.String("nodeID", event.GetID()))
				sendCh <- event
			} else {
				a.Log.Debug("filtered out artifact event", zap.String("nodeID", event.GetID()), zap.ByteString("body", event.GetBody()))
			}
		}
	}
}

func (a *artifact) Stop() error {
	err := a.streamSignal.Stop()
	close(a.streamEvents)
	return err
}

// checks if the notification satisfies the signal
// 3 conditions must be met
// 1. notification bucket name must equal the S3 bucket
// 2. notification event name must equal the signal S3 event
// 3. notification object must pass the prefix and suffix string literals
func checkArtifactSignal(notification minio.NotificationEvent, signal v1alpha1.ArtifactSignal) bool {
	if signal.S3.Filter != nil {
		return notification.S3.Bucket.Name == signal.S3.Bucket &&
			notification.EventName == string(signal.S3.Event) &&
			strings.HasPrefix(notification.S3.Object.Key, signal.S3.Filter.Prefix) &&
			strings.HasSuffix(notification.S3.Object.Key, signal.S3.Filter.Suffix)
	}
	return notification.S3.Bucket.Name == signal.S3.Bucket &&
		notification.EventName == string(signal.S3.Event)
}
