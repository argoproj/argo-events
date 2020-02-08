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
	"encoding/json"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventListener implements Eventing for mqtt event source
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
}

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	channels := server.NewChannels()

	go server.HandleEventsFromEventSource(eventSource.Name, eventStream, channels, listener.Logger)

	defer func() {
		channels.Stop <- struct{}{}
	}()

	if err := listener.listenEvents(eventSource, channels); err != nil {
		listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to listen to events")
		return err
	}

	return nil
}

// listenEvents listens to events from a mqtt broker
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var mqttEventSource *v1alpha1.MQTTEventSource
	if err := yaml.Unmarshal(eventSource.Value, &mqttEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse event source %s", eventSource.Name)
	}

	logger.Infoln("setting up the message handler...")
	handler := func(c mqttlib.Client, msg mqttlib.Message) {
		eventData := &apicommon.MQTTEventData{
			Topic:     msg.Topic(),
			MessageId: int(msg.MessageID()),
			Body:      msg.Payload(),
		}
		eventBody, err := json.Marshal(eventData)
		if err != nil {
			logger.WithError(err).Errorln("failed to marshal the event data, rejecting the event...")
			return
		}
		logger.Infoln("dispatching event on the data channel...")
		channels.Data <- eventBody
	}

	logger.Infoln("setting up the mqtt broker client...")
	opts := mqttlib.NewClientOptions().AddBroker(mqttEventSource.URL).SetClientID(mqttEventSource.ClientId)

	var client mqttlib.Client

	logger.Infoln("connecting to mqtt broker...")
	if err := server.Connect(common.GetConnectionBackoff(mqttEventSource.ConnectionBackoff), func() error {
		client = mqttlib.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to the mqtt broker for event source %s", eventSource.Name)
	}

	logger.Info("subscribing to the topic...")
	if token := client.Subscribe(mqttEventSource.Topic, 0, handler); token.Wait() && token.Error() != nil {
		return errors.Wrapf(token.Error(), "failed to subscribe to the topic %s for event source %s", mqttEventSource.Topic, eventSource.Name)
	}

	<-channels.Done
	logger.Infoln("event source is stopped, unsubscribing the client...")

	token := client.Unsubscribe(mqttEventSource.Topic)
	if token.Error() != nil {
		logger.WithError(token.Error()).Error("failed to unsubscribe client")
	}

	return nil
}
