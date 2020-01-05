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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
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
	logger *logrus.Logger
}

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)

	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents listens events published by nsq
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer server.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var nsqEventSource *v1alpha1.NSQEventSource
	if err := yaml.Unmarshal(eventSource.Value, &nsqEventSource); err != nil {
		errorCh <- errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
		return
	}

	// Instantiate a consumer that will subscribe to the provided channel.
	logger.Infoln("creating a NSQ consumer")
	var consumer *nsq.Consumer
	config := nsq.NewConfig()
	if err := server.Connect(common.GetConnectionBackoff(nsqEventSource.ConnectionBackoff), func() error {
		var err error
		if consumer, err = nsq.NewConsumer(nsqEventSource.Topic, nsqEventSource.Channel, config); err != nil {
			return err
		}
		return nil
	}); err != nil {
		errorCh <- errors.Wrapf(err, "failed to create a new consumer for topic %s and channel %s", nsqEventSource.Topic, nsqEventSource.Channel)
		return
	}

	consumer.AddHandler(&messageHandler{dataCh: dataCh})

	err := consumer.ConnectToNSQLookupd(nsqEventSource.HostAddress)
	if err != nil {
		errorCh <- errors.Wrapf(err, "lookup failed for host %s", nsqEventSource.HostAddress)
		return
	}

	<-doneCh
	logger.Infoln("event source has stopped")
	consumer.Stop()
}

// HandleMessage implements the Handler interface.
func (h *messageHandler) HandleMessage(m *nsq.Message) error {
	h.logger.WithField("message-id", m.ID).Infoln("received a message. dispatching the message on data channel")
	h.dataCh <- m.Body
	return nil
}
