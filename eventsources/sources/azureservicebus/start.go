package azureservicebus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	servicebus "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
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
	EventSourceName            string
	EventName                  string
	AzureServiceBusEventSource v1alpha1.AzureServiceBusEventSource
	Metrics                    *metrics.Metrics
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
	return apicommon.AzureServiceBus
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	log.Info("started processing the Azure Service Bus event source...")
	defer sources.Recover(el.GetEventName())

	servicebusEventSource := &el.AzureServiceBusEventSource
	connStr, err := common.GetSecretFromVolume(servicebusEventSource.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to retrieve connection string from secret %s, %s", servicebusEventSource.ConnectionString.Name, err)
	}

	log.Info("connecting to the service bus...")
	client, err := servicebus.NewClientFromConnectionString(connStr, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to the service bus, %w", err)
	}

	var receiver *servicebus.Receiver

	if servicebusEventSource.QueueName != "" {
		log.Info("creating a queue receiver...")
		receiver, err = client.NewReceiverForQueue(servicebusEventSource.QueueName, &servicebus.ReceiverOptions{
			ReceiveMode: servicebus.ReceiveModeReceiveAndDelete,
		})
		if err != nil {
			return fmt.Errorf("failed to create a receiver for queue %s, %w", servicebusEventSource.QueueName, err)
		}
	} else {
		log.Info("creating a subscription receiver...")
		receiver, err = client.NewReceiverForSubscription(servicebusEventSource.TopicName, servicebusEventSource.SubscriptionName, &servicebus.ReceiverOptions{
			ReceiveMode: servicebus.ReceiveModeReceiveAndDelete,
		})
		if err != nil {
			return fmt.Errorf("failed to create a receiver for topic %s and subscription %s, %w", servicebusEventSource.TopicName, servicebusEventSource.SubscriptionName, err)
		}
	}

	if servicebusEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("context has been cancelled, stopping the Azure Service Bus event source...")
			if err := receiver.Close(ctx); err != nil {
				log.Errorw("failed to close the receiver", zap.Error(err))
			}
			return nil
		default:
			messages, err := receiver.ReceiveMessages(ctx, 1, nil)
			if err != nil {
				return fmt.Errorf("failed to receive messages, %w", err)
			}

			for _, message := range messages {
				if err := el.handleOne(servicebusEventSource, message, dispatch, log); err != nil {
					log.With("queue", servicebusEventSource.QueueName, "message_id", message.MessageID).Errorw("failed to process Azure Service Bus message", zap.Error(err))
					el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
					continue
				}
			}
		}
	}
}

func (el *EventListener) handleOne(servicebusEventSource *v1alpha1.AzureServiceBusEventSource, message *servicebus.ReceivedMessage, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	log.Infow("received the message", zap.Any("message-id", message.MessageID))
	log.Info("received a message from the service bus")
	eventData := &events.AzureServiceBusEventData{
		ApplicationProperties: message.ApplicationProperties,
		ContentType:           message.ContentType,
		CorrelationId:         message.CorrelationID,
		EnqueuedTime:          message.EnqueuedTime,
		MessageId:             message.MessageID,
		ReplyTo:               message.ReplyTo,
		SequenceNumber:        message.SequenceNumber,
		Subject:               message.Subject,
		Metadata:              servicebusEventSource.Metadata,
	}

	if servicebusEventSource.JSONBody {
		eventData.Body = (*json.RawMessage)(&message.Body)
	} else {
		eventData.Body = message.Body
	}

	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		return fmt.Errorf("failed to marshal the event data for event source %s and message id %s, %w", el.GetEventName(), message.MessageID, err)
	}

	log.Info("dispatching the event to eventbus...")
	if err = dispatch(eventBytes); err != nil {
		el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		log.Errorw("failed to dispatch Azure Service Bus event", zap.Error(err))
		return err
	}

	return nil
}
