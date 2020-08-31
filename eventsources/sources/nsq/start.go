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
	"go.uber.org/zap"
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
	logger   *zap.SugaredLogger
	isJSON   bool
	metadata map[string]string
}

// StartListening listens NSQ events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the NSQ event source...")
	defer sources.Recover(el.GetEventName())

	nsqEventSource := &el.NSQEventSource

	// Instantiate a consumer that will subscribe to the provided channel.
	log.Info("creating a NSQ consumer")
	var consumer *nsq.Consumer
	config := nsq.NewConfig()

	if nsqEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(nsqEventSource.TLS)
		if err != nil {
			return errors.Wrap(err, "failed to get the tls configuration")
		}
		config.TlsConfig = tlsConfig
		config.TlsV1 = true
	}

	if err := common.Connect(common.GetConnectionBackoff(nsqEventSource.ConnectionBackoff), func() error {
		var err error
		if consumer, err = nsq.NewConsumer(nsqEventSource.Topic, nsqEventSource.Channel, config); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to create a new consumer for topic %s and channel %s for event source %s", nsqEventSource.Topic, nsqEventSource.Channel, el.GetEventName())
	}

	if nsqEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	consumer.AddHandler(&messageHandler{dispatch: dispatch, logger: log, isJSON: nsqEventSource.JSONBody, metadata: nsqEventSource.Metadata})

	err := consumer.ConnectToNSQLookupd(nsqEventSource.HostAddress)
	if err != nil {
		return errors.Wrapf(err, "lookup failed for host %s for event source %s", nsqEventSource.HostAddress, el.GetEventName())
	}

	<-ctx.Done()
	log.Info("event source has stopped")
	consumer.Stop()
	return nil
}

// HandleMessage implements the Handler interface.
func (h *messageHandler) HandleMessage(m *nsq.Message) error {
	h.logger.Info("received a message")

	eventData := &events.NSQEventData{
		Body:        m.Body,
		Timestamp:   strconv.Itoa(int(m.Timestamp)),
		NSQDAddress: m.NSQDAddress,
		Metadata:    h.metadata,
	}
	if h.isJSON {
		eventData.Body = (*json.RawMessage)(&m.Body)
	}

	eventBody, err := json.Marshal(eventData)
	if err != nil {
		h.logger.Desugar().Error("failed to marshal the event data. rejecting the event...", zap.Error(err))
		return err
	}

	h.logger.Info("dispatching the event on the data channel...")
	err = h.dispatch(eventBody)
	if err != nil {
		return err
	}
	return nil
}
