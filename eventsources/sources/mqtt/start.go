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

package mqtt

import (
	"context"
	"encoding/json"
	"time"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
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

// EventListener implements Eventing for mqtt event source
type EventListener struct {
	EventSourceName string
	EventName       string
	MQTTEventSource v1alpha1.MQTTEventSource
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
	return apicommon.MQTTEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	log.Info("starting MQTT event source...")
	mqttEventSource := &el.MQTTEventSource

	if mqttEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	log.Info("setting up the message handler...")
	handler := func(c mqttlib.Client, msg mqttlib.Message) {
		startTime := time.Now()
		defer func(start time.Time) {
			elapsed := time.Now().Sub(start)
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(elapsed/time.Millisecond))
		}(startTime)

		eventData := &events.MQTTEventData{
			Topic:     msg.Topic(),
			MessageID: int(msg.MessageID()),
			Body:      msg.Payload(),
			Metadata:  mqttEventSource.Metadata,
		}
		if mqttEventSource.JSONBody {
			body := msg.Payload()
			eventData.Body = (*json.RawMessage)(&body)
		} else {
			eventData.Body = msg.Payload()
		}

		eventBody, err := json.Marshal(eventData)
		if err != nil {
			log.Errorw("failed to marshal the event data, rejecting the event...", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			return
		}
		log.Info("dispatching event on the data channel...")
		if err = dispatch(eventBody); err != nil {
			log.Errorw("failed to dispatch MQTT event...", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		}
	}

	log.Info("setting up the mqtt broker client...")
	opts := mqttlib.NewClientOptions().AddBroker(mqttEventSource.URL).SetClientID(mqttEventSource.ClientID)
	if mqttEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(mqttEventSource.TLS)
		if err != nil {
			return errors.Wrap(err, "failed to get the tls configuration")
		}
		opts.TLSConfig = tlsConfig
	}

	var client mqttlib.Client

	log.Info("connecting to mqtt broker...")
	if err := common.Connect(mqttEventSource.ConnectionBackoff, func() error {
		client = mqttlib.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to the mqtt broker for event source %s", el.GetEventName())
	}

	log.Info("subscribing to the topic...")
	if token := client.Subscribe(mqttEventSource.Topic, 0, handler); token.Wait() && token.Error() != nil {
		return errors.Wrapf(token.Error(), "failed to subscribe to the topic %s for event source %s", mqttEventSource.Topic, el.GetEventName())
	}

	<-ctx.Done()
	log.Info("event source is stopped, unsubscribing the client...")

	token := client.Unsubscribe(mqttEventSource.Topic)
	if token.Error() != nil {
		log.Errorw("failed to unsubscribe client", zap.Error(token.Error()))
	}

	return nil
}
