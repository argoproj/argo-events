package aws_sqs

import (
	"fmt"
	"time"

	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/store"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	sqslib "github.com/aws/aws-sdk-go/service/sqs"
	cronlib "github.com/robfig/cron"
)

// Next is a function to compute the next event time from a given time
type Next func(time.Time) time.Time

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

func resolveSchedule(s *sqs) (cronlib.Schedule, error) {
	intervalDuration, err := time.ParseDuration(s.Frequency)
	if err != nil {
		return nil, fmt.Errorf("failed to parse interval %s. Cause: %+v", s.Frequency, err.Error())
	}
	schedule := cronlib.ConstantDelaySchedule{Delay: intervalDuration}
	return schedule, nil
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

	schedule, err := resolveSchedule(s)
	if err != nil {
		errorCh <- err
		return
	}

	var next Next
	next = func(last time.Time) time.Time {
		return schedule.Next(last)
	}

	lastT := time.Now()

	for {
		t := next(lastT)
		timer := time.After(time.Until(t))
		ese.Log.Info().Str("event-source-name", eventSource.Name).Str("time", t.UTC().String()).Msg("expected time to consume item from queue")
		select {
		case tx := <-timer:
			lastT = tx
			msg, err := sqsClient.ReceiveMessage(&sqslib.ReceiveMessageInput{
				QueueUrl:            queueURL.QueueUrl,
				MaxNumberOfMessages: aws.Int64(1),
			})
			if err != nil {
				ese.Log.Warn().Err(err).Str("event-source-name", eventSource.Name).Msg("failed to process item from queue, waiting for next interval")
				continue
			}
			dataCh <- []byte(*msg.Messages[0].Body)
			if s.DeleteAfterProcess {

			}
			sqsClient.DeleteMessage()
		case <-doneCh:
			return
		}
	}
}
