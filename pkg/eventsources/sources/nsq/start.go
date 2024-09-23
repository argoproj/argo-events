/*
Copyright 2020 The Argoproj Authors.

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
	"fmt"
	"strconv"
	"time"

	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for the NSQ event source
type EventListener struct {
	EventSourceName string
	EventName       string
	NSQEventSource  v1alpha1.NSQEventSource
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
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.NSQEvent
}

type messageHandler struct {
	eventSourceName string
	eventName       string
	metrics         *metrics.Metrics
	dispatch        func([]byte, ...eventsourcecommon.Option) error
	logger          *zap.SugaredLogger
	isJSON          bool
	metadata        map[string]string
}

// StartListening listens NSQ events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
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
		tlsConfig, err := sharedutil.GetTLSConfig(nsqEventSource.TLS)
		if err != nil {
			return fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		config.TlsConfig = tlsConfig
		config.TlsV1 = true
	}

	if err := sharedutil.DoWithRetry(nsqEventSource.ConnectionBackoff, func() error {
		var err error
		if consumer, err = nsq.NewConsumer(nsqEventSource.Topic, nsqEventSource.Channel, config); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create a new consumer for topic %s and channel %s for event source %s, %w", nsqEventSource.Topic, nsqEventSource.Channel, el.GetEventName(), err)
	}

	if nsqEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	consumer.AddHandler(&messageHandler{eventSourceName: el.EventSourceName, eventName: el.EventName, dispatch: dispatch, logger: log, isJSON: nsqEventSource.JSONBody, metadata: nsqEventSource.Metadata, metrics: el.Metrics})

	if err := consumer.ConnectToNSQLookupd(nsqEventSource.HostAddress); err != nil {
		return fmt.Errorf("lookup failed for host %s for event source %s, %w", nsqEventSource.HostAddress, el.GetEventName(), err)
	}

	<-ctx.Done()
	log.Info("event source has stopped")
	consumer.Stop()
	return nil
}

// HandleMessage implements the Handler interface.
func (h *messageHandler) HandleMessage(m *nsq.Message) error {
	defer func(start time.Time) {
		h.metrics.EventProcessingDuration(h.eventSourceName, h.eventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

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
		h.logger.Errorw("failed to marshal the event data. rejecting the event...", zap.Error(err))
		h.metrics.EventProcessingFailed(h.eventSourceName, h.eventName)
		return err
	}

	h.logger.Info("dispatching the event on the data channel...")
	if err = h.dispatch(eventBody); err != nil {
		h.metrics.EventProcessingFailed(h.eventSourceName, h.eventName)
		return err
	}
	return nil
}
