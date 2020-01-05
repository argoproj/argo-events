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

package nats

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	natslib "github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
)

// EventListener implements Eventing for nats event source
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
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

// listenEvents listens events from nats cluster
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer server.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var natsEventSource *v1alpha1.NATSEventsSource
	if err := yaml.Unmarshal(eventSource.Value, &natsEventSource); err != nil {
		errorCh <- err
		return
	}

	logger = logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: eventSource.Name,
			common.LabelURL:         natsEventSource.URL,
			"subject":               natsEventSource.Subject,
		},
	)

	var conn *natslib.Conn

	logger.Infoln("connecting to nats cluster...")
	if err := server.Connect(common.GetConnectionBackoff(natsEventSource.ConnectionBackoff), func() error {
		var err error
		if conn, err = natslib.Connect(natsEventSource.URL); err != nil {
			return err
		}
		return nil
	}); err != nil {
		logger.WithError(err).Error("failed to connect to nats cluster")
		errorCh <- err
		return
	}

	logger.Info("subscribing to messages on the queue...")
	_, err := conn.Subscribe(natsEventSource.Subject, func(msg *natslib.Msg) {
		logger.Infoln("dispatching event on data channel...")
		dataCh <- msg.Data
	})

	if err != nil {
		logger.WithError(err).Error("failed to subscribe")
		errorCh <- err
		return
	}

	conn.Flush()
	if err := conn.LastError(); err != nil {
		errorCh <- err
		return
	}

	<-doneCh
}
