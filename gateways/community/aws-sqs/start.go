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

package aws_sqs

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/store"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	sqslib "github.com/aws/aws-sdk-go/service/sqs"
)

// StartEventSource starts an event source
func (ese *SQSEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("activating event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		ese.Log.Error().Err(err).Str("event-source-name", eventSource.Name).Msg("failed to parse event source")

		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(config.(*sqs), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
}

// listenEvents fires an event when interval completes and item is processed from queue.
func (ese *SQSEventSourceExecutor) listenEvents(s *sqs, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	// retrieve access key id and secret access key
	accessKey, err := store.GetSecrets(ese.Clientset, ese.Namespace, s.AccessKey.Name, s.AccessKey.Key)
	if err != nil {
		errorCh <- err
		return
	}
	secretKey, err := store.GetSecrets(ese.Clientset, ese.Namespace, s.SecretKey.Name, s.SecretKey.Key)
	if err != nil {
		errorCh <- err
		return
	}

	creds := credentials.NewStaticCredentialsFromCreds(credentials.Value{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})
	awsSession, err := session.NewSession(&aws.Config{
		Region:      &s.Region,
		Credentials: creds,
	})
	if err != nil {
		errorCh <- err
		return
	}

	sqsClient := sqslib.New(awsSession)

	queueURL, err := sqsClient.GetQueueUrl(&sqslib.GetQueueUrlInput{
		QueueName: &s.Queue,
	})
	if err != nil {
		errorCh <- err
		return
	}

	for {
		select {
		case <-doneCh:
			return

		default:
			msg, err := sqsClient.ReceiveMessage(&sqslib.ReceiveMessageInput{
				QueueUrl:            queueURL.QueueUrl,
				MaxNumberOfMessages: aws.Int64(1),
				WaitTimeSeconds:     aws.Int64(s.WaitTimeSeconds),
			})
			if err != nil {
				ese.Log.Warn().Err(err).Str("event-source-name", eventSource.Name).Msg("failed to process item from queue, waiting for next timeout")
				continue
			}

			if msg != nil && len(msg.Messages) > 0 {
				dataCh <- []byte(*msg.Messages[0].Body)

				if _, err := sqsClient.DeleteMessage(&sqslib.DeleteMessageInput{
					QueueUrl:      queueURL.QueueUrl,
					ReceiptHandle: msg.Messages[0].ReceiptHandle,
				}); err != nil {
					errorCh <- err
					return
				}
			}
		}
	}
}
