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

package awssqs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	sqslib "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/argoproj/argo-events/common/logging"
	awscommon "github.com/argoproj/argo-events/eventsources/common/aws"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for aws sqs event source
type EventListener struct {
	EventSourceName string
	EventName       string
	SQSEventSource  v1alpha1.SQSEventSource
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
	return apicommon.SQSEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	log.Infoln("started processing the AWS SQS event source...")
	defer sources.Recover(el.GetEventName())

	sqsEventSource := &el.SQSEventSource
	var awsSession *session.Session
	awsSession, err := awscommon.CreateAWSSessionWithCredsInEnv(sqsEventSource.Region, sqsEventSource.RoleARN, sqsEventSource.AccessKey, sqsEventSource.SecretKey)
	if err != nil {
		return errors.Wrapf(err, "failed to create aws session for %s", el.GetEventName())
	}

	sqsClient := sqslib.New(awsSession)

	log.Infoln("fetching queue url...")
	getQueueURLInput := &sqslib.GetQueueUrlInput{
		QueueName: &sqsEventSource.Queue,
	}
	if sqsEventSource.QueueAccountID != "" {
		getQueueURLInput = getQueueURLInput.SetQueueOwnerAWSAccountId(sqsEventSource.QueueAccountID)
	}

	queueURL, err := sqsClient.GetQueueUrl(getQueueURLInput)
	if err != nil {
		return errors.Wrapf(err, "failed to get the queue url for %s", el.GetEventName())
	}

	if sqsEventSource.JSONBody {
		log.Infoln("assuming all events have a json body...")
	}

	log.Infoln("listening for messages on the queue...")
	for {
		select {
		case <-stopCh:
			log.Info("exiting SQS event listener...")
			return nil
		default:
		}
		messages, err := fetchMessages(ctx, sqsClient, *queueURL.QueueUrl, 10, sqsEventSource.WaitTimeSeconds)
		if err != nil {
			log.WithError(err).Errorln("failed to get messages from SQS")
			time.Sleep(2 * time.Second)
			continue
		}
		for _, m := range messages {
			el.processMessage(ctx, m, dispatch, func() {
				_, err = sqsClient.DeleteMessage(&sqslib.DeleteMessageInput{
					QueueUrl:      queueURL.QueueUrl,
					ReceiptHandle: m.ReceiptHandle,
				})
				if err != nil {
					log.WithError(err).Errorln("Failed to delete message")
				}
			}, log)
		}
	}
}

func (el *EventListener) processMessage(ctx context.Context, message *sqslib.Message, dispatch func([]byte) error, ack func(), log *logrus.Entry) {
	data := &events.SQSEventData{
		MessageId:         *message.MessageId,
		MessageAttributes: message.MessageAttributes,
	}
	if el.SQSEventSource.JSONBody {
		body := []byte(*message.Body)
		data.Body = (*json.RawMessage)(&body)
	} else {
		data.Body = []byte(*message.Body)
	}
	eventBytes, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Errorln("failed to marshal event data, will process next message...")
		ack()
		return
	}
	err = dispatch(eventBytes)
	if err != nil {
		log.WithError(err).Errorln("failed to dispatch event")
	} else {
		ack()
	}
}

func fetchMessages(ctx context.Context, q *sqslib.SQS, url string, maxSize, waitSeconds int64) ([]*sqslib.Message, error) {
	if waitSeconds == 0 {
		// Defaults to 3 seconds
		waitSeconds = 3
	}
	result, err := q.ReceiveMessageWithContext(ctx, &sqslib.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqslib.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqslib.QueueAttributeNameAll),
		},
		QueueUrl:            &url,
		MaxNumberOfMessages: aws.Int64(maxSize),
		VisibilityTimeout:   aws.Int64(120), // 120 seconds
		WaitTimeSeconds:     aws.Int64(waitSeconds),
	})
	if err != nil {
		return nil, err
	}
	return result.Messages, nil
}
