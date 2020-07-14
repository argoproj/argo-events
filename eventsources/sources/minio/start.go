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

package minio

import (
	"context"
	"encoding/json"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/minio/minio-go"
	"github.com/pkg/errors"
)

// MinioEventSourceListener implements Eventing for minio event sources
type EventListener struct {
	EventSourceName  string
	EventName        string
	MinioEventSource apicommon.S3Artifact
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.MinioEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	defer sources.Recover(el.GetEventName())

	log.Infoln("starting minio event source...")

	minioEventSource := &el.MinioEventSource

	log.Info("retrieving access and secret key...")
	accessKey, ok := common.GetEnvFromSecret(minioEventSource.AccessKey)
	if !ok {
		return errors.Errorf("failed to get the access key for event source %s in ENV", el.GetEventName())
	}
	secretKey, ok := common.GetEnvFromSecret(minioEventSource.SecretKey)
	if !ok {
		return errors.Errorf("failed to retrieve the secret key for event source %s in ENV", el.GetEventName())
	}

	log.Infoln("setting up a minio client...")
	minioClient, err := minio.New(minioEventSource.Endpoint, accessKey, secretKey, !minioEventSource.Insecure)
	if err != nil {
		return errors.Wrapf(err, "failed to create a client for event source %s", el.GetEventName())
	}

	prefix, suffix := getFilters(minioEventSource)

	doneCh := make(chan struct{})

	log.WithField("bucket-name", minioEventSource.Bucket.Name).Info("started listening to bucket notifications...")
	for notification := range minioClient.ListenBucketNotification(minioEventSource.Bucket.Name, prefix, suffix, minioEventSource.Events, doneCh) {
		if notification.Err != nil {
			log.WithError(notification.Err).Errorln("invalid notification")
			continue
		}

		eventData := &events.MinioEventData{Notification: notification.Records}
		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			log.WithError(notification.Err).Errorln("failed to marshal the event data, rejecting the event...")
			continue
		}

		log.Infoln("dispatching the event on data channel...")
		if err = dispatch(eventBytes); err != nil {
			log.WithError(err).Errorln("failed to dispatch event")
		}
	}

	<-stopCh
	doneCh <- struct{}{}

	log.Infoln("event source is stopped")
	return nil
}

func getFilters(eventSource *apicommon.S3Artifact) (string, string) {
	if eventSource.Filter == nil {
		return "", ""
	}
	if eventSource.Filter.Prefix != "" && eventSource.Filter.Suffix != "" {
		return eventSource.Filter.Prefix, eventSource.Filter.Suffix
	}
	if eventSource.Filter.Prefix != "" {
		return eventSource.Filter.Prefix, ""
	}
	if eventSource.Filter.Suffix != "" {
		return "", eventSource.Filter.Suffix
	}
	return "", ""
}
