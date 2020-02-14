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
	"encoding/json"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	natslib "github.com/nats-io/go-nats"
	"github.com/pkg/errors"
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

// listenEvents listens events from nats cluster
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var natsEventSource *v1alpha1.NATSEventsSource
	if err := yaml.Unmarshal(eventSource.Value, &natsEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse event source %s", eventSource.Name)
	}

	var conn *natslib.Conn

	logger.Infoln("connecting to nats cluster...")
	if err := server.Connect(common.GetConnectionBackoff(natsEventSource.ConnectionBackoff), func() error {
		var err error
		if conn, err = natslib.Connect(natsEventSource.URL); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to the nats server for event source %s", eventSource.Name)
	}

	logger.Info("subscribing to messages on the queue...")
	_, err := conn.Subscribe(natsEventSource.Subject, func(msg *natslib.Msg) {
		eventData := &apicommon.NATSEventData{
			Subject: msg.Subject,
			Body:    msg.Data,
		}
		eventBody, err := json.Marshal(eventData)
		if err != nil {
			logger.WithError(err).Errorln("failed to marshal the event data, rejecting the event...")
			return
		}
		logger.Infoln("dispatching the event on data channel...")
		channels.Data <- eventBody
	})
	if err != nil {
		return errors.Wrapf(err, "failed to subscribe to the subject %s for event source %s", natsEventSource.Subject, eventSource.Name)
	}

	conn.Flush()
	if err := conn.LastError(); err != nil {
		return errors.Wrapf(err, "connection failure for event source %s", eventSource.Name)
	}

	<-channels.Done
	logger.Infoln("event source is stopped")
	return nil
}
