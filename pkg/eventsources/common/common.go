package common

import (
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	natslib "github.com/nats-io/nats.go"
)

type Option func(*event.Event) error

// WithID sets a different ID for the event.
func WithID(id string) Option {
	return func(e *event.Event) error {
		e.SetID(id)
		return nil
	}
}

// WithSource overrides the event source attribute.
func WithSource(source string) Option {
	return func(e *event.Event) error {
		e.SetSource(source)
		return nil
	}
}

// WithType overrides the event type attribute.
func WithType(ceType string) Option {
	return func(e *event.Event) error {
		e.SetType(ceType)
		return nil
	}
}

// WithSubject overrides the event subject attribute.
func WithSubject(subject string) Option {
	return func(e *event.Event) error {
		e.SetSubject(subject)
		return nil
	}
}

// WithTime overrides the event time attribute.
func WithTime(t time.Time) Option {
	return func(e *event.Event) error {
		e.SetTime(t)
		return nil
	}
}

// WithExtension sets a CloudEvent extension attribute.
func WithExtension(key string, value interface{}) Option {
	return func(e *event.Event) error {
		e.SetExtension(key, value)
		return nil
	}
}

// WithCloudEvent applies attributes and extensions from an incoming CloudEvent.
// Only non-zero/non-empty attributes from the incoming event override the defaults.
// All extensions from the incoming event are copied.
func WithCloudEvent(incoming cloudevents.Event) Option {
	return func(e *event.Event) error {
		if incoming.ID() != "" {
			e.SetID(incoming.ID())
		}
		if incoming.Source() != "" {
			e.SetSource(incoming.Source())
		}
		if incoming.Type() != "" {
			e.SetType(incoming.Type())
		}
		if incoming.Subject() != "" {
			e.SetSubject(incoming.Subject())
		}
		if !incoming.Time().IsZero() {
			e.SetTime(incoming.Time())
		}
		if incoming.DataContentType() != "" {
			e.SetDataContentType(incoming.DataContentType())
		}
		for k, v := range incoming.Extensions() {
			e.SetExtension(k, v)
		}
		return nil
	}
}

// WithHTTPHeaders extracts W3C trace context headers (traceparent/tracestate)
// from HTTP request headers and sets them as CloudEvent extensions.
// This enables trace propagation for non-CloudEvent HTTP requests.
func WithHTTPHeaders(headers http.Header) Option {
	return func(e *event.Event) error {
		if tp := headers.Get("Traceparent"); tp != "" {
			e.SetExtension("traceparent", tp)
		}
		if ts := headers.Get("Tracestate"); ts != "" {
			e.SetExtension("tracestate", ts)
		}
		return nil
	}
}

// WithKafkaHeaders extracts W3C trace context headers (traceparent/tracestate)
// from a Kafka record's headers map and sets them as CloudEvent extensions.
// This enables trace propagation from Kafka producers (e.g. franz-go via
// otel.GetTextMapPropagator().Inject) into the eventbus dispatch chain
// where SpanFromCloudEvent will pick them up as the parent span.
func WithKafkaHeaders(headers map[string]string) Option {
	return func(e *event.Event) error {
		if tp, ok := headers["traceparent"]; ok && tp != "" {
			e.SetExtension("traceparent", tp)
		}
		if ts, ok := headers["tracestate"]; ok && ts != "" {
			e.SetExtension("tracestate", ts)
		}
		return nil
	}
}

// WithNATSHeaders extracts W3C trace context headers (traceparent/tracestate)
// from a NATS message's headers and sets them as CloudEvent extensions.
// This enables trace propagation from NATS/JetStream producers (e.g. nats.go
// via otel.GetTextMapPropagator().Inject into nats.Header) into the eventbus
// dispatch chain where SpanFromCloudEvent will pick them up as the parent
// span.
//
// nats.Header.Set does NOT canonicalize keys (unlike http.Header), so
// lookups are done case-sensitively for both lowercase ("traceparent",
// emitted by OTel propagators) and Title case ("Traceparent", emitted
// by callers using textproto helpers). Safe to call with a nil header.
func WithNATSHeaders(headers natslib.Header) Option {
	return func(e *event.Event) error {
		if headers == nil {
			return nil
		}
		if tp := firstNonEmpty(headers, "traceparent", "Traceparent"); tp != "" {
			e.SetExtension("traceparent", tp)
		}
		if ts := firstNonEmpty(headers, "tracestate", "Tracestate"); ts != "" {
			e.SetExtension("tracestate", ts)
		}
		return nil
	}
}

func firstNonEmpty(h natslib.Header, keys ...string) string {
	for _, k := range keys {
		if v := h.Get(k); v != "" {
			return v
		}
	}
	return ""
}
