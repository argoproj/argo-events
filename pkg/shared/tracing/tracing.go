package tracing

import (
	"context"
	"fmt"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer initializes OpenTelemetry tracing using standard OTel environment variables:
//   - OTEL_EXPORTER_OTLP_ENDPOINT: The OTLP collector endpoint (e.g., "http://jaeger:4317")
//   - OTEL_TRACES_EXPORTER: Set to "none" to disable tracing (default: "otlp")
//   - OTEL_SERVICE_NAME: Overrides the serviceName parameter
//
// When OTEL_EXPORTER_OTLP_ENDPOINT is not set, tracing is disabled (no-op).
// Returns a shutdown function that should be deferred.
func InitTracer(serviceName string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }

	// Check if tracing is explicitly disabled
	if os.Getenv("OTEL_TRACES_EXPORTER") == "none" {
		return noop, nil
	}

	// If no endpoint is configured, use no-op
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return noop, nil
	}

	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return noop, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return noop, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// w3cCarrier adapts CloudEvent extensions to the TextMapCarrier interface
// for W3C Trace Context propagation.
type w3cCarrier struct {
	event *cloudevents.Event
}

func (c *w3cCarrier) Get(key string) string {
	v := c.event.Extensions()[key]
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func (c *w3cCarrier) Set(key, value string) {
	c.event.SetExtension(key, value)
}

func (c *w3cCarrier) Keys() []string {
	exts := c.event.Extensions()
	keys := make([]string, 0, len(exts))
	for k := range exts {
		keys = append(keys, k)
	}
	return keys
}

// SpanFromCloudEvent extracts the trace context (traceparent/tracestate) from
// CloudEvent extensions and returns a context with the remote span context attached.
// If no trace context is present, returns the original context unchanged.
func SpanFromCloudEvent(ctx context.Context, event cloudevents.Event) context.Context {
	carrier := &w3cCarrier{event: &event}
	prop := otel.GetTextMapPropagator()
	return prop.Extract(ctx, carrier)
}

// InjectTraceIntoCloudEvent writes the current span's trace context (traceparent/tracestate)
// into the CloudEvent's extensions. If no active span exists, this is a no-op.
func InjectTraceIntoCloudEvent(ctx context.Context, event *cloudevents.Event) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	carrier := &w3cCarrier{event: event}
	prop := otel.GetTextMapPropagator()
	prop.Inject(ctx, carrier)
}
