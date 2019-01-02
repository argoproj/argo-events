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
	"github.com/minio/minio-go"
)

// StartEventSource activates an event source and streams back events
func (ce *S3ConfigExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ce.GatewayConfig.Log.Info().Str("event-source-name", eventSource.Name).Msg("activating event source")
	artifact, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}
	ce.GatewayConfig.Log.Debug().Str("event-source-name", eventSource.Name).Interface("event-source-value", *artifact).Msg("artifact event source")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{})

	go ce.listenToEvents(artifact, dataCh, errorCh, doneCh)

	for {
		select {
		case data := <-dataCh:
			err := eventStream.Send(&gateways.Event{
				Name: eventSource.Name,
				Payload: data,
			})
			if err != nil {
				return err
			}

		case err := <-errorCh:
			return err

		case <-eventStream.Context().Done():
			ce.Log.Info().Str("event-source-name", eventSource.Name).Msg("connection is closed by client")
			return nil
		}
	}
}

// listenEvents listens to minio bucket notifications
func (ce *S3ConfigExecutor) listenToEvents(artifact *S3Artifact, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	// retrieve access key id and secret access key
	accessKey, err := gateways.GetSecret(ce.Clientset, ce.Namespace, artifact.AccessKey.Name, artifact.AccessKey.Key)
	if err != nil {
		errorCh <- err
		return
	}
	secretKey, err := gateways.GetSecret(ce.Clientset, ce.Namespace, artifact.SecretKey.Name, artifact.SecretKey.Key)
	if err != nil {
		errorCh <- err
		return
	}

	minioClient, err := minio.New(artifact.S3EventConfig.Endpoint, accessKey, secretKey, !artifact.Insecure)
	if err != nil {
		errorCh <- err
		return
	}

	for notification := range minioClient.ListenBucketNotification(artifact.S3EventConfig.Bucket, artifact.S3EventConfig.Filter.Prefix, artifact.S3EventConfig.Filter.Suffix, []string{
		string(artifact.S3EventConfig.Event),
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
