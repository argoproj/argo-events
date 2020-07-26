package pulsar

import (
	"context"
	"encoding/json"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// EventListener implements Eventing for the Pulsar event source
type EventListener struct {
	EventSourceName   string
	EventName         string
	PulsarEventSource v1alpha1.PulsarEventSource
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
	return apicommon.PulsarEvent
}

// StartListening listens NSQ events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the Pulsar event source...")
	defer sources.Recover(el.GetEventName())

	msgChannel := make(chan pulsar.ConsumerMessage)

	pulsarEventSource := &el.PulsarEventSource

	subscriptionType := pulsar.Exclusive
	if pulsarEventSource.Type == "shared" {
		subscriptionType = pulsar.Shared
	}

	log.Info("setting consumer options...")
	consumerOpt := pulsar.ConsumerOptions{
		Topics:           pulsarEventSource.Topics,
		SubscriptionName: el.EventName,
		Type:             subscriptionType,
		MessageChannel:   msgChannel,
	}

	log.Info("setting client options...")
	clientOpt := pulsar.ClientOptions{
		URL:                        pulsarEventSource.URL,
		TLSTrustCertsFilePath:      pulsarEventSource.TLSTrustCertsFilePath,
		TLSAllowInsecureConnection: pulsarEventSource.TLSAllowInsecureConnection,
		TLSValidateHostname:        pulsarEventSource.TLSValidateHostname,
	}

	if pulsarEventSource.TLS != nil {
		log.Info("setting tls auth option...")
		clientOpt.Authentication = pulsar.NewAuthenticationTLS(pulsarEventSource.TLS.ClientCertPath, pulsarEventSource.TLS.ClientKeyPath)
	}

	var client pulsar.Client

	if err := sources.Connect(common.GetConnectionBackoff(pulsarEventSource.ConnectionBackoff), func() error {
		var err error
		if client, err = pulsar.NewClient(clientOpt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to %s for event source %s", pulsarEventSource.URL, el.GetEventName())
	}

	consumer, err := client.Subscribe(consumerOpt)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to topic %+v for event source %s", pulsarEventSource.Topics, el.GetEventName())
	}

consumeMessages:
	for {
		select {
		case msg := <-msgChannel:
			payload := msg.Payload()
			eventData := &events.PulsarEventData{
				Key:         msg.Key(),
				PublishTime: msg.PublishTime().UTC().String(),
				Body:        payload,
			}
			if pulsarEventSource.JSONBody {
				eventData.Body = (*json.RawMessage)(&payload)
			}

			eventBody, err := json.Marshal(eventData)
			if err != nil {
				log.Error("failed to marshal the event data. rejecting the event...", zap.Error(err))
				return err
			}

			err = dispatch(eventBody)
			if err != nil {
				log.Error("failed to dispatch NATS event", zap.Error(err))
			}

		case <-ctx.Done():
			consumer.Close()
			client.Close()
			break consumeMessages
		}
	}

	log.Info("event source is stopped")
	return nil
}
