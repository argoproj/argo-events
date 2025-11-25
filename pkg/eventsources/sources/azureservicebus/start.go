package azureservicebus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	servicebus "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"go.uber.org/zap"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for azure events hub event source
type EventListener struct {
	EventSourceName            string
	EventName                  string
	AzureServiceBusEventSource aev1.AzureServiceBusEventSource
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
func (el *EventListener) GetEventSourceType() aev1.EventSourceType {
	return aev1.AzureServiceBus
}

type ReceiverType string

const (
	ReceiverTypeQueue        ReceiverType = "queue"
	ReceiverTypeSubscription ReceiverType = "subscription"
)

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	log.Info("started processing the Azure Service Bus event source...")
	defer sources.Recover(el.GetEventName())

	servicebusEventSource := &el.AzureServiceBusEventSource
	clientOptions := servicebus.ClientOptions{}
	if servicebusEventSource.TLS != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(servicebusEventSource.TLS)
		if err != nil {
			log.Errorw("failed to get the tls configuration", zap.Error(err))
			return err
		}
		clientOptions.TLSConfig = tlsConfig
	}
	var client *servicebus.Client
	if servicebusEventSource.ConnectionString != nil {
		log.Info("connecting to the service bus using connection string...")
		connStr, err := sharedutil.GetSecretFromVolume(servicebusEventSource.ConnectionString)
		if err != nil {
			log.With("connection-string", servicebusEventSource.ConnectionString.Name).Errorw("failed to retrieve connection string from secret", zap.Error(err))
			return err
		}
		client, err = servicebus.NewClientFromConnectionString(connStr, &clientOptions)
		if err != nil {
			log.Errorw("failed to create a service bus client", zap.Error(err))
			return err
		}
	} else {
		log.Info("connecting to azure queue storage with AAD credentials...")
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			log.Errorw("failed to create DefaultAzureCredential", zap.Error(err))
			return err
		}
		client, err = servicebus.NewClient(servicebusEventSource.FullyQualifiedNamespace, cred, &clientOptions)
		if err != nil {
			log.Errorw("failed to create a service bus client", zap.Error(err))
			return err
		}
	}

	var receiverOptions = servicebus.ReceiverOptions{ReceiveMode: servicebus.ReceiveModeReceiveAndDelete}
	if servicebusEventSource.DeferDelete {
		receiverOptions.ReceiveMode = servicebus.ReceiveModePeekLock
	}

	var receiver *servicebus.Receiver
	var receiverType ReceiverType
	var err error

	if servicebusEventSource.QueueName != "" {
		log.Info("creating a queue receiver...")
		receiverType = ReceiverTypeQueue
		receiver, err = client.NewReceiverForQueue(servicebusEventSource.QueueName, &receiverOptions)
	} else {
		log.Info("creating a subscription receiver...")
		receiverType = ReceiverTypeSubscription
		receiver, err = client.NewReceiverForSubscription(servicebusEventSource.TopicName, servicebusEventSource.SubscriptionName, &receiverOptions)
	}
	if err != nil {
		if receiverType == ReceiverTypeQueue {
			log.With("queue", servicebusEventSource.QueueName).Errorw("failed to create a queue receiver", zap.Error(err))
		} else {
			log.With("topic", servicebusEventSource.TopicName, "subscription", servicebusEventSource.SubscriptionName).Errorw("failed to create a receiver for subscription", zap.Error(err))
		}
		return err
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
				return err
			}
			return nil
		default:
			messages, err := receiver.ReceiveMessages(ctx, 1, nil)
			if err != nil {
				log.Errorw("failed to receive messages", zap.Error(err))
				continue
			}

			for _, message := range messages {
				if err := el.handleOne(servicebusEventSource, message, dispatch, log); err != nil {
					if receiverType == ReceiverTypeQueue {
						log.With("queue", servicebusEventSource.QueueName, "message_id", message.MessageID).Errorw("failed to process Azure Service Bus message", zap.Error(err))
					} else {
						log.With("topic", servicebusEventSource.TopicName, "subscription", servicebusEventSource.SubscriptionName, "message_id", message.MessageID).Errorw("failed to process Azure Service Bus message", zap.Error(err))
					}
					el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
					continue
				}
				if servicebusEventSource.DeferDelete {
					err = receiver.CompleteMessage(ctx, message, nil)
					if err != nil {
						log.With("message_id", message.MessageID).Errorw("failed to complete message", zap.Error(err))
						continue
					}
				}
			}
		}
	}
}

func (el *EventListener) handleOne(servicebusEventSource *aev1.AzureServiceBusEventSource, message *servicebus.ReceivedMessage, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	log.Infow("received the message", zap.Any("message_id", message.MessageID))
	eventData := &events.AzureServiceBusEventData{
		ApplicationProperties: message.ApplicationProperties,
		ContentType:           message.ContentType,
		CorrelationID:         message.CorrelationID,
		EnqueuedTime:          message.EnqueuedTime,
		MessageID:             message.MessageID,
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
		log.With("event_source", el.GetEventSourceName(), "event", el.GetEventName(), "message_id", message.MessageID).Errorw("failed to marshal the event data", zap.Error(err))
		return err
	}

	log.Info("dispatching the event to eventbus...")
	if err = dispatch(eventBytes); err != nil {
		log.With("event_source", el.GetEventSourceName(), "event", el.GetEventName(), "message_id", message.MessageID).Errorw("failed to dispatch the event", zap.Error(err))
		return err
	}

	return nil
}
