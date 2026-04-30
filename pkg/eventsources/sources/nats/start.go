/*
Copyright 2018 The Argoproj Authors.

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
	"fmt"
	"time"

	natslib "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	"github.com/argoproj/argo-events/pkg/shared/tracing"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// natsHeaderCarrier adapts nats.Header to propagation.TextMapCarrier so the
// OTel TextMapPropagator can Extract/Inject W3C trace context across the
// NATS subscribe boundary. nats.Header.Set is a plain map write (no
// canonicalisation), so producers and consumers must agree on case;
// callers that follow the standard W3C casing (lowercase "traceparent")
// interoperate with OTel's TraceContext propagator out of the box.
type natsHeaderCarrier natslib.Header

func (c natsHeaderCarrier) Get(key string) string { return natslib.Header(c).Get(key) }
func (c natsHeaderCarrier) Set(key, val string)   { natslib.Header(c).Set(key, val) }
func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// EventListener implements Eventing for nats event source
type EventListener struct {
	EventSourceName string
	EventName       string
	NATSEventSource v1alpha1.NATSEventsSource
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
	return v1alpha1.NATSEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	natsEventSource := &el.NATSEventSource

	var opt []natslib.Option
	if natsEventSource.TLS != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(natsEventSource.TLS)
		if err != nil {
			return fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		opt = append(opt, natslib.Secure(tlsConfig))
	}

	if natsEventSource.Auth != nil {
		switch {
		case natsEventSource.Auth.Basic != nil:
			username, err := sharedutil.GetSecretFromVolume(natsEventSource.Auth.Basic.Username)
			if err != nil {
				return err
			}
			password, err := sharedutil.GetSecretFromVolume(natsEventSource.Auth.Basic.Password)
			if err != nil {
				return err
			}
			opt = append(opt, natslib.UserInfo(username, password))
		case natsEventSource.Auth.Token != nil:
			token, err := sharedutil.GetSecretFromVolume(natsEventSource.Auth.Token)
			if err != nil {
				return err
			}
			opt = append(opt, natslib.Token(token))
		case natsEventSource.Auth.NKey != nil:
			nkeyFile, err := sharedutil.GetSecretVolumePath(natsEventSource.Auth.NKey)
			if err != nil {
				return err
			}
			o, err := natslib.NkeyOptionFromSeed(nkeyFile)
			if err != nil {
				return fmt.Errorf("failed to get NKey, %w", err)
			}
			opt = append(opt, o)
		case natsEventSource.Auth.Credential != nil:
			cFile, err := sharedutil.GetSecretVolumePath(natsEventSource.Auth.Credential)
			if err != nil {
				return err
			}
			opt = append(opt, natslib.UserCredentials(cFile))
		}
	}

	var conn *natslib.Conn
	log.Info("connecting to nats cluster...")
	if err := sharedutil.DoWithRetry(natsEventSource.ConnectionBackoff, func() error {
		var err error
		if conn, err = natslib.Connect(natsEventSource.URL, opt...); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to connect to the nats server for event source %s, %w", el.GetEventName(), err)
	}
	defer conn.Close()

	if natsEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	handler := func(msg *natslib.Msg) {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		// Ensure we have a header map even if the message arrived without one;
		// the inject step below needs a writable carrier.
		if msg.Header == nil {
			msg.Header = natslib.Header{}
		}

		// Extract upstream W3C trace context from the NATS message headers.
		parentCtx := otel.GetTextMapPropagator().Extract(ctx, natsHeaderCarrier(msg.Header))

		// Start an eventsource.consume CONSUMER span as the parent for the
		// downstream eventsource.publish PRODUCER span emitted in eventing.go.
		spanCtx, consumeSpan := tracing.StartConsumerSpan(parentCtx, otel.Tracer("argo-events-eventsource"), "eventsource.consume",
			attribute.String("eventsource.name", el.GetEventSourceName()),
			attribute.String("eventsource.type", "nats"),
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", msg.Subject),
			attribute.String("messaging.operation.type", "receive"),
		)
		defer consumeSpan.End()

		// Re-inject the CONSUMER span's trace context into the headers so
		// WithNATSHeaders writes the CONSUMER traceparent as the CloudEvent
		// extension. eventing.go's SpanFromCloudEvent then makes the
		// eventsource.publish PRODUCER span a child of CONSUMER.
		otel.GetTextMapPropagator().Inject(spanCtx, natsHeaderCarrier(msg.Header))

		eventData := &events.NATSEventData{
			Subject:  msg.Subject,
			Header:   msg.Header,
			Metadata: natsEventSource.Metadata,
		}
		if natsEventSource.JSONBody {
			eventData.Body = (*json.RawMessage)(&msg.Data)
		} else {
			eventData.Body = msg.Data
		}

		eventBody, err := json.Marshal(eventData)
		if err != nil {
			log.Errorw("failed to marshal the event data, rejecting the event...", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			consumeSpan.RecordError(err)
			consumeSpan.SetStatus(codes.Error, err.Error())
			return
		}
		log.Info("dispatching the event on data channel...")
		if err = dispatch(eventBody, eventsourcecommon.WithNATSHeaders(msg.Header)); err != nil {
			log.Errorw("failed to dispatch a NATS event", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			consumeSpan.RecordError(err)
			consumeSpan.SetStatus(codes.Error, err.Error())
		}
	}

	var err error
	if natsEventSource.Queue != nil {
		log.Infof("subscribing to messages on the subject %s queue %s", natsEventSource.Subject, *natsEventSource.Queue)
		_, err = conn.QueueSubscribe(natsEventSource.Subject, *natsEventSource.Queue, handler)
	} else {
		log.Infof("subscribing to messages on the subject %s", natsEventSource.Subject)
		_, err = conn.Subscribe(natsEventSource.Subject, handler)
	}

	if err != nil {
		return fmt.Errorf("failed to subscribe to the subject %s for event source %s, %w", natsEventSource.Subject, el.GetEventName(), err)
	}

	conn.Flush()
	if err := conn.LastError(); err != nil {
		return fmt.Errorf("connection failure for event source %s, %w", el.GetEventName(), err)
	}

	<-ctx.Done()
	log.Info("event source is stopped")
	return nil
}
