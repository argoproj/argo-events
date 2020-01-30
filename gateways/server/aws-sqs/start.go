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
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	commonaws "github.com/argoproj/argo-events/gateways/server/common/aws"
	common2 "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	sqslib "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for aws sqs event source
type EventListener struct {
	Logger *logrus.Logger
	// k8sClient is kubernetes client
	K8sClient kubernetes.Interface
}

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer server.Recover(eventSource.Name)

	log := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("started processing the event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)
	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents fires an event when interval completes and item is processed from queue.
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	var sqsEventSource *v1alpha1.SQSEventSource
	if err := yaml.Unmarshal(eventSource.Value, &sqsEventSource); err != nil {
		errorCh <- err
		return
	}

	var awsSession *session.Session

	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("setting up aws session...")
	awsSession, err := commonaws.CreateAWSSession(listener.K8sClient, sqsEventSource.Namespace, sqsEventSource.Region, sqsEventSource.AccessKey, sqsEventSource.SecretKey)
	if err != nil {
		errorCh <- err
		return
	}

	sqsClient := sqslib.New(awsSession)

	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("fetching queue url...")
	queueURL, err := sqsClient.GetQueueUrl(&sqslib.GetQueueUrlInput{
		QueueName: &sqsEventSource.Queue,
	})
	if err != nil {
		errorCh <- err
		return
	}

	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("listening for messages on the queue...")
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
				listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Error("failed to process item from queue, waiting for next timeout")
				continue
			}

			if msg != nil && len(msg.Messages) > 0 {
				message := *msg.Messages[0]

				listener.Logger.WithFields(map[string]interface{}{
					common.LabelEventSource: eventSource.Name,
					"message":               message.Body,
				}).Debugln("message from queue")

				data := &common2.SQSEventData{
					MessageId:         *message.MessageId,
					MessageAttributes: message.MessageAttributes,
					Body:              []byte(*message.Body),
				}

				eventBytes, err := json.Marshal(data)
				if err != nil {
					listener.Logger.WithError(err).WithField(common.LabelEventSource, eventSource.Name).Errorln("failed to marshal event data")
					continue
				}

				listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("dispatching the event on data channel")
				dataCh <- eventBytes

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
