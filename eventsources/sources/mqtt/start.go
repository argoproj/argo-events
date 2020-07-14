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

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for mqtt event source
type EventListener struct {
	EventSourceName string
	EventName       string
	MQTTEventSource v1alpha1.MQTTEventSource
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
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	defer sources.Recover(el.GetEventName())

	log.Infoln("starting MQTT event source...")
	mqttEventSource := &el.MQTTEventSource

	if mqttEventSource.JSONBody {
		log.Infoln("assuming all events have a json body...")
	}

	log.Infoln("setting up the message handler...")
	handler := func(c mqttlib.Client, msg mqttlib.Message) {
		eventData := &events.MQTTEventData{
			Topic:     msg.Topic(),
			MessageID: int(msg.MessageID()),
			Body:      msg.Payload(),
		}
		if mqttEventSource.JSONBody {
			body := msg.Payload()
			eventData.Body = (*json.RawMessage)(&body)
		} else {
			eventData.Body = msg.Payload()
		}

		eventBody, err := json.Marshal(eventData)
		if err != nil {
			log.WithError(err).Errorln("failed to marshal the event data, rejecting the event...")
			return
		}
		log.Infoln("dispatching event on the data channel...")
		if err = dispatch(eventBody); err != nil {
			log.WithError(err).Errorln("failed to dispatch event...")
		}
	}

	log.Infoln("setting up the mqtt broker client...")
	opts := mqttlib.NewClientOptions().AddBroker(mqttEventSource.URL).SetClientID(mqttEventSource.ClientID)
	if mqttEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(mqttEventSource.TLS.CACertPath, mqttEventSource.TLS.ClientCertPath, mqttEventSource.TLS.ClientKeyPath)
		if err != nil {
			return errors.Wrap(err, "failed to get the tls configuration")
		}
		opts.TLSConfig = tlsConfig
	}

	var client mqttlib.Client

	log.Infoln("connecting to mqtt broker...")
	if err := sources.Connect(common.GetConnectionBackoff(mqttEventSource.ConnectionBackoff), func() error {
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

	<-stopCh
	log.Infoln("event source is stopped, unsubscribing the client...")

	token := client.Unsubscribe(mqttEventSource.Topic)
	if token.Error() != nil {
		log.WithError(token.Error()).Error("failed to unsubscribe client")
	}

	return nil
}
