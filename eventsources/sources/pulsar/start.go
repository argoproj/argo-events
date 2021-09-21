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
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for the Pulsar event source
type EventListener struct {
	EventSourceName   string
	EventName         string
	PulsarEventSource v1alpha1.PulsarEventSource
	Metrics           *metrics.Metrics
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
	var err error
	tlsTrustCertsFilePath := ""
	if pulsarEventSource.TLSTrustCertsSecret != nil {
		tlsTrustCertsFilePath, err = common.GetSecretVolumePath(pulsarEventSource.TLSTrustCertsSecret)
		if err != nil {
			log.Errorw("failed to get TLSTrustCertsFilePath from the volume", "error", err)
			return err
		}
	}
	clientOpt := pulsar.ClientOptions{
		URL:                        pulsarEventSource.URL,
		TLSTrustCertsFilePath:      tlsTrustCertsFilePath,
		TLSAllowInsecureConnection: pulsarEventSource.TLSAllowInsecureConnection,
		TLSValidateHostname:        pulsarEventSource.TLSValidateHostname,
	}

	if pulsarEventSource.TLS != nil {
		log.Info("setting tls auth option...")
		var clientCertPath, clientKeyPath string
		switch {
		case pulsarEventSource.TLS.ClientCertSecret != nil && pulsarEventSource.TLS.ClientKeySecret != nil:
			clientCertPath, err = common.GetSecretVolumePath(pulsarEventSource.TLS.ClientCertSecret)
			if err != nil {
				log.Errorw("failed to get ClientCertPath from the volume", "error", err)
				return err
			}
			clientKeyPath, err = common.GetSecretVolumePath(pulsarEventSource.TLS.ClientKeySecret)
			if err != nil {
				log.Errorw("failed to get ClientKeyPath from the volume", "error", err)
				return err
			}
		default:
			return errors.New("invalid TLS config")
		}
		clientOpt.Authentication = pulsar.NewAuthenticationTLS(clientCertPath, clientKeyPath)
	}

	var client pulsar.Client

	if err := common.Connect(pulsarEventSource.ConnectionBackoff, func() error {
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
		case msg, ok := <-msgChannel:
			if !ok {
				log.Error("failed to read a message, channel might have been closed")
				return errors.New("channel might have been closed")
			}

			if err := el.handleOne(msg, dispatch, log); err != nil {
				log.Errorw("failed to process a Pulsar event", zap.Error(err))
				el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
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

func (el *EventListener) handleOne(msg pulsar.Message, dispatch func([]byte) error, log *zap.SugaredLogger) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	log.Infof("received a message on the topic %s", msg.Topic())
	payload := msg.Payload()
	eventData := &events.PulsarEventData{
		Key:         msg.Key(),
		PublishTime: msg.PublishTime().UTC().String(),
		Body:        payload,
		Metadata:    el.PulsarEventSource.Metadata,
	}
	if el.PulsarEventSource.JSONBody {
		eventData.Body = (*json.RawMessage)(&payload)
	}

	eventBody, err := json.Marshal(eventData)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the event data. rejecting the event...")
	}

	log.Infof("dispatching the message received on the topic %s to eventbus", msg.Topic())
	if err = dispatch(eventBody); err != nil {
		return errors.Wrap(err, "failed to dispatch a Pulsar event")
	}
	return nil
}
