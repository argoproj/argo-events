package azurequeuestorage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for azure events hub event source
type EventListener struct {
	EventSourceName              string
	EventName                    string
	AzureQueueStorageEventSource v1alpha1.AzureQueueStorageEventSource
	Metrics                      *metrics.Metrics
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
	return apicommon.AzureQueueStorage
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	log.Info("started processing the Azure Queue Storage event source...")
	defer sources.Recover(el.GetEventName())

	queueStorageEventSource := &el.AzureQueueStorageEventSource
	var client *azqueue.ServiceClient
	// if connectionString is set then use it
	// otherwise try to connect via Azure Active Directory (AAD) with storageAccountName
	if queueStorageEventSource.ConnectionString != nil {
		connStr, err := common.GetSecretFromVolume(queueStorageEventSource.ConnectionString)
		if err != nil {
			log.With("connection-string", queueStorageEventSource.ConnectionString.Name).Errorw("failed to retrieve connection string from secret", zap.Error(err))
			return err
		}

		log.Info("connecting to azure queue storage with connection string...")
		client, err = azqueue.NewServiceClientFromConnectionString(connStr, nil)
		if err != nil {
			log.Errorw("failed to create a service client", zap.Error(err))
			return err
		}
	} else {
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			log.Errorw("failed to create DefaultAzureCredential", zap.Error(err))
			return err
		}
		log.Info("connecting to azure queue storage with AAD credentials...")
		serviceURL := fmt.Sprintf("https://%s.queue.core.windows.net/", queueStorageEventSource.StorageAccountName)
		client, err = azqueue.NewServiceClient(serviceURL, cred, nil)
		if err != nil {
			log.Errorw("failed to create a service client", zap.Error(err))
			return err
		}
	}

	queueClient := client.NewQueueClient(el.AzureQueueStorageEventSource.QueueName)
	if queueStorageEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}
	var numMessages int32 = 10
	var visibilityTimeout int32 = 120
	log.Info("listening for messages on the queue...")
	for {
		select {
		case <-ctx.Done():
			log.Info("exiting AQS event listener...")
			return nil
		default:
		}
		messages, err := queueClient.DequeueMessages(ctx, &azqueue.DequeueMessagesOptions{
			NumberOfMessages:  &numMessages,
			VisibilityTimeout: &visibilityTimeout,
		})
		if err != nil {
			log.Errorw("failed to get messages from AQS", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}
		for _, m := range messages.Messages {
			el.processMessage(m, dispatch, func() {
				_, err = queueClient.DeleteMessage(ctx, *m.MessageID, *m.PopReceipt, &azqueue.DeleteMessageOptions{})
				if err != nil {
					log.Errorw("Failed to delete message", zap.Error(err))
				}
			}, log)
		}
	}
}

func (el *EventListener) processMessage(message *azqueue.DequeuedMessage, dispatch func([]byte, ...eventsourcecommon.Option) error, ack func(), log *zap.SugaredLogger) {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())
	data := &events.AzureQueueStorageEventData{
		MessageID:     *message.MessageID,
		InsertionTime: *message.InsertionTime,
		Metadata:      el.AzureQueueStorageEventSource.Metadata,
	}
	if el.AzureQueueStorageEventSource.JSONBody {
		body := []byte(*message.MessageText)
		data.Body = (*json.RawMessage)(&body)
	} else {
		data.Body = []byte(*message.MessageText)
	}
	eventBytes, err := json.Marshal(data)
	if err != nil {
		log.Errorw("failed to marshal event data, will process next message...", zap.Error(err))
		el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		// Don't ack if a DLQ is configured to allow to forward the message to the DLQ
		if !el.AzureQueueStorageEventSource.DLQ {
			ack()
		}
		return
	}
	if err = dispatch(eventBytes); err != nil {
		log.Errorw("failed to dispatch azure queue storage event", zap.Error(err))
		el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
	} else {
		ack()
	}
}
