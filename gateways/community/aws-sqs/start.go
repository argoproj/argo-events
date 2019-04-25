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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/aws/aws-sdk-go/aws"
	sqslib "github.com/aws/aws-sdk-go/service/sqs"
)

// StartEventSource starts an event source
func (ese *SQSEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	log := ese.Log.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("activating event source")

	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(config.(*sqsEventSource), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, ese.Log)
}

// listenEvents fires an event when interval completes and item is processed from queue.
func (ese *SQSEventSourceExecutor) listenEvents(s *sqsEventSource, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	creds, err := gwcommon.GetAWSCreds(ese.Clientset, ese.Namespace, s.AccessKey, s.SecretKey)
	if err != nil {
		errorCh <- err
		return
	}

	awsSession, err := gwcommon.GetAWSSession(creds, s.Region)
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
				ese.Log.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Error("failed to process item from queue, waiting for next timeout")
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
