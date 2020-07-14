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
	"context"
	"encoding/json"
	"strconv"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/nsqio/go-nsq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventListener implements Eventing for the NSQ event source
type EventListener struct {
	EventSourceName string
	EventName       string
	NSQEventSource  v1alpha1.NSQEventSource
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
	return apicommon.NSQEvent
}

type messageHandler struct {
	dispatch func([]byte) error
	logger   *logrus.Entry
	isJSON   bool
}

// StartListening listens NSQ events
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	log.Infoln("started processing the NSQ event source...")
	defer sources.Recover(el.GetEventName())

	nsqEventSource := &el.NSQEventSource

	// Instantiate a consumer that will subscribe to the provided channel.
	log.Infoln("creating a NSQ consumer")
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

	if err := sources.Connect(common.GetConnectionBackoff(nsqEventSource.ConnectionBackoff), func() error {
		var err error
		if consumer, err = nsq.NewConsumer(nsqEventSource.Topic, nsqEventSource.Channel, config); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to create a new consumer for topic %s and channel %s for event source %s", nsqEventSource.Topic, nsqEventSource.Channel, el.GetEventName())
	}

	if nsqEventSource.JSONBody {
		log.Infoln("assuming all events have a json body...")
	}

	consumer.AddHandler(&messageHandler{dispatch: dispatch, logger: log, isJSON: nsqEventSource.JSONBody})

	err := consumer.ConnectToNSQLookupd(nsqEventSource.HostAddress)
	if err != nil {
		return errors.Wrapf(err, "lookup failed for host %s for event source %s", nsqEventSource.HostAddress, el.GetEventName())
	}

	<-stopCh
	log.Infoln("event source has stopped")
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
	err = h.dispatch(eventBody)
	if err != nil {
		return err
	}
	return nil
}
