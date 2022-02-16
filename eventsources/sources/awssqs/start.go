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
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	awscommon "github.com/argoproj/argo-events/eventsources/common/aws"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for aws sqs event source
type EventListener struct {
	EventSourceName string
	EventName       string
	SQSEventSource  v1alpha1.SQSEventSource
	Metrics         *metrics.Metrics
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
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Options) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the AWS SQS event source...")
	defer sources.Recover(el.GetEventName())

	sqsEventSource := &el.SQSEventSource
	var awsSession *session.Session
	awsSession, err := awscommon.CreateAWSSessionWithCredsInVolume(sqsEventSource.Region, sqsEventSource.RoleARN, sqsEventSource.AccessKey, sqsEventSource.SecretKey)
	if err != nil {
		log.Errorw("Error creating AWS credentials", zap.Error(err))
		return errors.Wrapf(err, "failed to create aws session for %s", el.GetEventName())
	}

	var sqsClient *sqslib.SQS

	if sqsEventSource.Endpoint == "" {
		sqsClient = sqslib.New(awsSession)
	} else {
		sqsClient = sqslib.New(awsSession, &aws.Config{Endpoint: &sqsEventSource.Endpoint, Region: &sqsEventSource.Region})
	}

	log.Info("fetching queue url...")
	getQueueURLInput := &sqslib.GetQueueUrlInput{
		QueueName: &sqsEventSource.Queue,
	}
	if sqsEventSource.QueueAccountID != "" {
		getQueueURLInput = getQueueURLInput.SetQueueOwnerAWSAccountId(sqsEventSource.QueueAccountID)
	}

	queueURL, err := sqsClient.GetQueueUrl(getQueueURLInput)
	if err != nil {
		log.Errorw("Error getting SQS Queue URL", zap.Error(err))
		return errors.Wrapf(err, "failed to get the queue url for %s", el.GetEventName())
	}

	if sqsEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	log.Info("listening for messages on the queue...")
	for {
		select {
		case <-ctx.Done():
			log.Info("exiting SQS event listener...")
			return nil
		default:
		}
		messages, err := fetchMessages(ctx, sqsClient, *queueURL.QueueUrl, 10, sqsEventSource.WaitTimeSeconds)
		if err != nil {
			log.Errorw("failed to get messages from SQS", zap.Error(err))
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
					log.Errorw("Failed to delete message", zap.Error(err))
				}
			}, log)
		}
	}
}

func (el *EventListener) processMessage(ctx context.Context, message *sqslib.Message, dispatch func([]byte, ...eventsourcecommon.Options) error, ack func(), log *zap.SugaredLogger) {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	data := &events.SQSEventData{
		MessageId:         *message.MessageId,
		MessageAttributes: message.MessageAttributes,
		Metadata:          el.SQSEventSource.Metadata,
	}
	if el.SQSEventSource.JSONBody {
		body := []byte(*message.Body)
		data.Body = (*json.RawMessage)(&body)
	} else {
		data.Body = []byte(*message.Body)
	}
	eventBytes, err := json.Marshal(data)
	if err != nil {
		log.Errorw("failed to marshal event data, will process next message...", zap.Error(err))
		el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		// Don't ack if a DLQ is configured to allow to forward the message to the DLQ
		if !el.SQSEventSource.DLQ {
			ack()
		}
		return
	}
	if err = dispatch(eventBytes); err != nil {
		log.Errorw("failed to dispatch SQS event", zap.Error(err))
		el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
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
