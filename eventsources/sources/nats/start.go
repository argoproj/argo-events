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
	"context"
	"encoding/json"

	natslib "github.com/nats-io/go-nats"
	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for nats event source
type EventListener struct {
	EventSourceName string
	EventName       string
	NATSEventSource v1alpha1.NATSEventsSource
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
	return apicommon.NATSEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	defer sources.Recover(el.GetEventName())

	natsEventSource := &el.NATSEventSource

	var conn *natslib.Conn

	log.Infoln("connecting to nats cluster...")
	if err := sources.Connect(common.GetConnectionBackoff(natsEventSource.ConnectionBackoff), func() error {
		var err error
		var opt []natslib.Option

		if natsEventSource.TLS != nil {
			tlsConfig, err := common.GetTLSConfig(natsEventSource.TLS.CACertPath, natsEventSource.TLS.ClientCertPath, natsEventSource.TLS.ClientKeyPath)
			if err != nil {
				return errors.Wrap(err, "failed to get the tls configuration")
			}
			opt = append(opt, natslib.Secure(tlsConfig))
		}

		if conn, err = natslib.Connect(natsEventSource.URL, opt...); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to the nats server for event source %s", el.GetEventName())
	}

	if natsEventSource.JSONBody {
		log.Infoln("assuming all events have a json body...")
	}

	log.Info("subscribing to messages on the queue...")
	_, err := conn.Subscribe(natsEventSource.Subject, func(msg *natslib.Msg) {
		eventData := &events.NATSEventData{
			Subject: msg.Subject,
		}
		if natsEventSource.JSONBody {
			eventData.Body = (*json.RawMessage)(&msg.Data)
		} else {
			eventData.Body = msg.Data
		}

		eventBody, err := json.Marshal(eventData)
		if err != nil {
			log.WithError(err).Errorln("failed to marshal the event data, rejecting the event...")
			return
		}
		log.Infoln("dispatching the event on data channel...")
		err = dispatch(eventBody)
		if err != nil {
			log.WithError(err).Errorln("failed to dispatch event")
		}
	})
	if err != nil {
		return errors.Wrapf(err, "failed to subscribe to the subject %s for event source %s", natsEventSource.Subject, el.GetEventName())
	}

	conn.Flush()
	if err := conn.LastError(); err != nil {
		return errors.Wrapf(err, "connection failure for event source %s", el.GetEventName())
	}

	<-stopCh
	log.Infoln("event source is stopped")
	return nil
}
