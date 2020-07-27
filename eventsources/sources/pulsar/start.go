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

// StartListening listens Pulsar events
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

	log.Info("subscribing to messages on the topic...")
	consumer, err := client.Subscribe(consumerOpt)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to topic %+v for event source %s", pulsarEventSource.Topics, el.GetEventName())
	}

consumeMessages:
	for {
		select {
		case msg := <-msgChannel:
			log.Infof("received a message on the topic %s", msg.Topic())
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
				log.Desugar().Error("failed to marshal the event data. rejecting the event...", zap.Error(err))
				return err
			}

			log.Infof("dispatching the message received on the topic %s to eventbus", msg.Topic())
			err = dispatch(eventBody)
			if err != nil {
				log.Desugar().Error("failed to dispatch Pulsar event", zap.Error(err))
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
