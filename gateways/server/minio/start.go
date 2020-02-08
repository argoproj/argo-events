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
	"encoding/json"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/store"
	"github.com/ghodss/yaml"
	"github.com/minio/minio-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// MinioEventSourceListener implements Eventing for minio event sources
type EventListener struct {
	// Logger
	Logger *logrus.Logger
	// K8sClient is kubernetes client
	K8sClient kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// StartEventSource activates an event source and streams back events
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	channels := server.NewChannels()

	go server.HandleEventsFromEventSource(eventSource.Name, eventStream, channels, listener.Logger)

	defer func() {
		channels.Stop <- struct{}{}
	}()

	if err := listener.listenEvents(eventSource, channels); err != nil {
		listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to listen to events")
		return err
	}

	return nil
}

// listenEvents listens to minio bucket notifications
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing minio event source...")
	var minioEventSource *apicommon.S3Artifact
	err := yaml.Unmarshal(eventSource.Value, &minioEventSource)
	if err != nil {
		return errors.Wrapf(err, "failed to parse event source %s", eventSource.Name)
	}

	logger.Info("retrieving access and secret key...")
	accessKey, err := store.GetSecrets(listener.K8sClient, listener.Namespace, minioEventSource.AccessKey.Name, minioEventSource.AccessKey.Key)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve the access key for event source %s", eventSource.Name)
	}
	secretKey, err := store.GetSecrets(listener.K8sClient, listener.Namespace, minioEventSource.SecretKey.Name, minioEventSource.SecretKey.Key)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve the secret key for event source %s", eventSource.Name)
	}

	logger.Infoln("setting up a minio client...")
	minioClient, err := minio.New(minioEventSource.Endpoint, accessKey, secretKey, !minioEventSource.Insecure)
	if err != nil {
		return errors.Wrapf(err, "failed to create a client for event source %s", eventSource.Name)
	}

	prefix, suffix := getFilters(minioEventSource)

	doneCh := make(chan struct{})

	logger.WithField("bucket-name", minioEventSource.Bucket.Name).Info("started listening to bucket notifications...")
	for notification := range minioClient.ListenBucketNotification(minioEventSource.Bucket.Name, prefix, suffix, minioEventSource.Events, doneCh) {
		if notification.Err != nil {
			logger.WithError(notification.Err).Errorln("invalid notification")
			continue
		}

		eventData := &apicommon.MinioEventData{Notification: notification.Records}
		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			logger.WithError(notification.Err).Errorln("failed to marshal the event data, rejecting the event...")
			continue
		}

		logger.Infoln("dispatching the event on data channel...")
		channels.Data <- eventBytes
	}

	<-channels.Done
	doneCh <- struct{}{}

	logger.Infoln("event source is stopped")
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
