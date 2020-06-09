/*
Copyright 2020 BlackRock, Inc.

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

package nsq

import (
	"encoding/json"
	"strconv"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventListener implements Eventing for the NSQ event source
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
}

type messageHandler struct {
	dataCh chan []byte
	logger *logrus.Entry
	isJSON bool
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

// listenEvents listens events published by nsq
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var nsqEventSource *v1alpha1.NSQEventSource
	if err := yaml.Unmarshal(eventSource.Value, &nsqEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
	}

	// Instantiate a consumer that will subscribe to the provided channel.
	logger.Infoln("creating a NSQ consumer")
	var consumer *nsq.Consumer
	config := nsq.NewConfig()

	if nsqEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(nsqEventSource.TLS.CACertPath, nsqEventSource.TLS.ClientCertPath, nsqEventSource.TLS.ClientKeyPath)
		if err != nil {
			return errors.Wrap(err, "failed to get the tls configuration")
		}
		config.TlsConfig = tlsConfig
		config.TlsV1 = true
	}

	if err := server.Connect(common.GetConnectionBackoff(nsqEventSource.ConnectionBackoff), func() error {
		var err error
		if consumer, err = nsq.NewConsumer(nsqEventSource.Topic, nsqEventSource.Channel, config); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to create a new consumer for topic %s and channel %s for event source %s", nsqEventSource.Topic, nsqEventSource.Channel, eventSource.Name)
	}

	if nsqEventSource.JSONBody {
		logger.Infoln("assuming all events have a json body...")
	}

	consumer.AddHandler(&messageHandler{dataCh: channels.Data, logger: logger, isJSON: nsqEventSource.JSONBody})

	err := consumer.ConnectToNSQLookupd(nsqEventSource.HostAddress)
	if err != nil {
		return errors.Wrapf(err, "lookup failed for host %s for event source %s", nsqEventSource.HostAddress, eventSource.Name)
	}

	<-channels.Done
	logger.Infoln("event source has stopped")
	consumer.Stop()
	return nil
}

// HandleMessage implements the Handler interface.
func (h *messageHandler) HandleMessage(m *nsq.Message) error {
	h.logger.Infoln("received a message")

	eventData := &events.NSQEventData{
		Body:        m.Body,
		Timestamp:   strconv.Itoa(int(m.Timestamp)),
		NSQDAddress: m.NSQDAddress,
	}
	if h.isJSON {
		eventData.Body = (*json.RawMessage)(&m.Body)
	}

	eventBody, err := json.Marshal(eventData)
	if err != nil {
		h.logger.WithError(err).Errorln("failed to marshal the event data. rejecting the event...")
		return err
	}

	h.logger.Infoln("dispatching the event on the data channel...")
	h.dataCh <- eventBody
	return nil
}
