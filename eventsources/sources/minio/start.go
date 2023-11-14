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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/notification"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
)

// EventListener implements Eventing for minio event sources
type EventListener struct {
	EventSourceName  string
	EventName        string
	MinioEventSource apicommon.S3Artifact
	Metrics          *metrics.Metrics
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
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName(),
			zap.String("bucketName", el.MinioEventSource.Bucket.Name))
	defer sources.Recover(el.GetEventName())

	log.Info("starting minio event source...")

	minioEventSource := &el.MinioEventSource

	log.Info("retrieving access and secret key...")
	accessKey, err := common.GetSecretFromVolume(minioEventSource.AccessKey)
	if err != nil {
		return fmt.Errorf("failed to get the access key for event source %s, %w", el.GetEventName(), err)
	}
	secretKey, err := common.GetSecretFromVolume(minioEventSource.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve the secret key for event source %s, %w", el.GetEventName(), err)
	}

	var minioClient *minio.Client
	var clientErr error
	if minioEventSource.CACertificate != nil {
		log.Info("retrieving CA certificate...")
		caCertificate, err := common.GetSecretFromVolume(minioEventSource.CACertificate)
		if err != nil {
			return fmt.Errorf("failed to get the CA certificate for event source %s, %w", el.GetEventName(), err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caCertificate))
		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		log.Info("setting up a minio client with custom CA...")
		minioClient, clientErr = minio.New(minioEventSource.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: !minioEventSource.Insecure,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		})
	} else {
		log.Info("setting up a minio client...")
		minioClient, clientErr = minio.New(minioEventSource.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: !minioEventSource.Insecure,
		})
	}
	if clientErr != nil {
		return fmt.Errorf("failed to create a client for event source %s, %w", el.GetEventName(), err)
	}

	prefix, suffix := getFilters(minioEventSource)

	log.Info("started listening to bucket notifications...")
	for notification := range minioClient.ListenBucketNotification(ctx, minioEventSource.Bucket.Name, prefix, suffix, minioEventSource.Events) {
		if notification.Err != nil {
			log.Errorw("invalid notification", zap.Error(notification.Err))
			continue
		}

		if err := el.handleOne(notification, dispatch, log); err != nil {
			log.Errorw("failed to process a Minio event", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		}
	}

	log.Info("event source is stopped")
	return nil
}

func (el *EventListener) handleOne(notification notification.Info, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	eventData := &events.MinioEventData{Notification: notification.Records, Metadata: el.MinioEventSource.Metadata}
	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal the event data, rejecting the event, %w", err)
	}

	log.Info("dispatching the event on data channel...")
	if err = dispatch(eventBytes); err != nil {
		return fmt.Errorf("failed to dispatch minio event, %w", err)
	}
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
