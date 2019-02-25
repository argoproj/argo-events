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

	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/store"
	"github.com/minio/minio-go"
)

// StartEventSource activates an event source and streams back events
func (ese *S3EventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("activating event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		ese.Log.Error().Err(err).Str("event-source-name", eventSource.Name).Msg("failed to parse event source")
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(config.(*apicommon.S3Artifact), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
}

// listenEvents listens to minio bucket notifications
func (ese *S3EventSourceExecutor) listenEvents(a *apicommon.S3Artifact, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Interface("event-source", eventSource).Msg("operating on event source")

	ese.Log.Info().Str("event-source-name", eventSource.Name).Interface("event-source", eventSource).Msg("retrieving access and secret key")
	// retrieve access key id and secret access key
	accessKey, err := store.GetSecrets(ese.Clientset, ese.Namespace, a.AccessKey.Name, a.AccessKey.Key)
	if err != nil {
		errorCh <- err
		return
	}
	secretKey, err := store.GetSecrets(ese.Clientset, ese.Namespace, a.SecretKey.Name, a.SecretKey.Key)
	if err != nil {
		errorCh <- err
		return
	}

	minioClient, err := minio.New(a.Endpoint, accessKey, secretKey, !a.Insecure)
	if err != nil {
		errorCh <- err
		return
	}

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("starting to listen to bucket notifications")
	for notification := range minioClient.ListenBucketNotification(a.Bucket.Name, a.Filter.Prefix, a.Filter.Suffix, []string{
		string(a.Event),
	}, doneCh) {
		if notification.Err != nil {
			errorCh <- notification.Err
			return
		}
		payload, err := json.Marshal(notification.Records[0])
		if err != nil {
			errorCh <- err
			return
		}
		dataCh <- payload
	}
}
