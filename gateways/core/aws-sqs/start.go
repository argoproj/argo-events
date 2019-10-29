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
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	sqslib "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// SQSEventSourceListener implements Eventing
type SQSEventSourceListener struct {
	Log *logrus.Logger
	// k8sClient is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// StartEventSource starts an event source
func (ese *SQSEventSourceListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	log := ese.Log.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("activating event source")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(eventSource, dataCh, errorCh, doneCh)
	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, ese.Log)
}

// listenEvents fires an event when interval completes and item is processed from queue.
func (ese *SQSEventSourceListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	var sqsEventSource *v1alpha1.SQSEventSource
	if err := yaml.Unmarshal(eventSource.Value, &sqsEventSource); err != nil {
		errorCh <- err
		return
	}

	var awsSession *session.Session

	if sqsEventSource.AccessKey == nil && sqsEventSource.SecretKey == nil {
		awsSessionWithoutCreds, err := gwcommon.GetAWSSessionWithoutCreds(sqsEventSource.Region)
		if err != nil {
			errorCh <- err
			return
		}

		awsSession = awsSessionWithoutCreds
	} else {
		creds, err := gwcommon.GetAWSCreds(ese.Clientset, ese.Namespace, sqsEventSource.AccessKey, sqsEventSource.SecretKey)
		if err != nil {
			errorCh <- err
			return
		}

		awsSessionWithCreds, err := gwcommon.GetAWSSession(creds, sqsEventSource.Region)
		if err != nil {
			errorCh <- err
			return
		}

		awsSession = awsSessionWithCreds
	}

	sqsClient := sqslib.New(awsSession)

	queueURL, err := sqsClient.GetQueueUrl(&sqslib.GetQueueUrlInput{
		QueueName: &sqsEventSource.Queue,
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
				WaitTimeSeconds:     aws.Int64(sqsEventSource.WaitTimeSeconds),
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
