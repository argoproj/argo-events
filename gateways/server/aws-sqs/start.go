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
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	sqslib "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for aws sqs event source
type EventListener struct {
	Logger *logrus.Logger
	// k8sClient is kubernetes client
	K8sClient kubernetes.Interface
	Namespace string
}

// StartEventSource starts an event source
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

// listenEvents fires an event when interval completes and item is processed from queue.
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var sqsEventSource *v1alpha1.SQSEventSource
	if err := yaml.Unmarshal(eventSource.Value, &sqsEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
	}

	if sqsEventSource.Namespace == "" {
		sqsEventSource.Namespace = listener.Namespace
	}

	logger.Infoln("setting up aws session...")

	var awsSession *session.Session

	awsSession, err := commonaws.CreateAWSSession(listener.K8sClient, sqsEventSource.Namespace, sqsEventSource.Region, sqsEventSource.RoleARN, sqsEventSource.AccessKey, sqsEventSource.SecretKey)

	if err != nil {
		return errors.Wrapf(err, "failed to create aws session for %s", eventSource.Name)
	}

	sqsClient := sqslib.New(awsSession)

	logger.Infoln("fetching queue url...")
	getQueueUrlInput := &sqslib.GetQueueUrlInput{
		QueueName: &sqsEventSource.Queue,
	}
	if sqsEventSource.QueueAccountId != "" {
		getQueueUrlInput = getQueueUrlInput.SetQueueOwnerAWSAccountId(sqsEventSource.QueueAccountId)
	}

	queueURL, err := sqsClient.GetQueueUrl(getQueueUrlInput)
	if err != nil {
		return errors.Wrapf(err, "failed to get the queue url for %s", eventSource.Name)
	}

	if sqsEventSource.JSONBody {
		logger.Infoln("assuming all events have a json body...")
	}

	logger.Infoln("listening for messages on the queue...")
	for {
		select {
		case <-channels.Done:
			return nil

		default:
			msg, err := sqsClient.ReceiveMessage(&sqslib.ReceiveMessageInput{
				QueueUrl:            queueURL.QueueUrl,
				MaxNumberOfMessages: aws.Int64(1),
				WaitTimeSeconds:     aws.Int64(sqsEventSource.WaitTimeSeconds),
			})
			if err != nil {
				logger.WithError(err).Errorln("failed to process item from queue, waiting for next timeout")
				continue
			}

			if msg != nil && len(msg.Messages) > 0 {
				message := *msg.Messages[0]

				data := &events.SQSEventData{
					MessageId:         *message.MessageId,
					MessageAttributes: message.MessageAttributes,
				}
				if sqsEventSource.JSONBody {
					body := []byte(*message.Body)
					data.Body = (*json.RawMessage)(&body)
				} else {
					data.Body = []byte(*message.Body)
				}

				eventBytes, err := json.Marshal(data)
				if err != nil {
					logger.WithError(err).Errorln("failed to marshal event data, will process next message...")
					continue
				}

				logger.Infoln("dispatching the event on data channel")
				channels.Data <- eventBytes

				if _, err := sqsClient.DeleteMessage(&sqslib.DeleteMessageInput{
					QueueUrl:      queueURL.QueueUrl,
					ReceiptHandle: msg.Messages[0].ReceiptHandle,
				}); err != nil {
					return errors.Wrapf(err, "failed to delete the message after consuming from queue for event source %s", eventSource.Name)
				}
			}
		}
	}
}
