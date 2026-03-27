package common

import (
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
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
