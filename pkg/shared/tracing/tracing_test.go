package tracing

import (
	"context"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestInitTracer_NoEndpoint(t *testing.T) {
	// When OTEL_EXPORTER_OTLP_ENDPOINT is not set, InitTracer returns no-op
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_TRACES_EXPORTER", "")

	shutdown, err := InitTracer("test-service")
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Shutdown should be a no-op and not error
	err = shutdown(context.Background())
	assert.NoError(t, err)
}

func TestInitTracer_Disabled(t *testing.T) {
	// When OTEL_TRACES_EXPORTER is "none", tracing is disabled
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")

	shutdown, err := InitTracer("test-service")
	require.NoError(t, err)

	err = shutdown(context.Background())
	assert.NoError(t, err)
}

func TestW3cCarrier(t *testing.T) {
	event := cloudevents.NewEvent()
	event.SetExtension("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	event.SetExtension("tracestate", "congo=t61rcWkgMzE")

	carrier := &w3cCarrier{event: &event}

	assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", carrier.Get("traceparent"))
	assert.Equal(t, "congo=t61rcWkgMzE", carrier.Get("tracestate"))
	assert.Equal(t, "", carrier.Get("nonexistent"))

	keys := carrier.Keys()
	assert.Contains(t, keys, "traceparent")
	assert.Contains(t, keys, "tracestate")

	carrier.Set("newkey", "newvalue")
	assert.Equal(t, "newvalue", carrier.Get("newkey"))
}

func TestSpanFromCloudEvent(t *testing.T) {
	// Set up a real propagator for the test
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Run("extracts trace context from CE extensions", func(t *testing.T) {
		event := cloudevents.NewEvent()
		event.SetExtension("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

		ctx := SpanFromCloudEvent(context.Background(), event)
		sc := trace.SpanContextFromContext(ctx)

		assert.True(t, sc.HasTraceID())
		assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", sc.TraceID().String())
		assert.Equal(t, "00f067aa0ba902b7", sc.SpanID().String())
		assert.True(t, sc.IsSampled())
	})

	t.Run("returns original context when no trace", func(t *testing.T) {
		event := cloudevents.NewEvent()

		ctx := SpanFromCloudEvent(context.Background(), event)
		sc := trace.SpanContextFromContext(ctx)

		assert.False(t, sc.HasTraceID())
	})
}

func TestInjectTraceIntoCloudEvent(t *testing.T) {
	// Set up a real tracer and propagator for the test
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() { _ = tp.Shutdown(context.Background()) }()

	t.Run("injects trace context into CE extensions", func(t *testing.T) {
		ctx, span := tp.Tracer("test").Start(context.Background(), "test-span")
		defer span.End()

		event := cloudevents.NewEvent()
		InjectTraceIntoCloudEvent(ctx, &event)

		// traceparent should be set
		tp := event.Extensions()["traceparent"]
		assert.NotNil(t, tp)
		assert.Contains(t, tp, span.SpanContext().TraceID().String())
	})

	t.Run("no-op when no active span", func(t *testing.T) {
		event := cloudevents.NewEvent()
		InjectTraceIntoCloudEvent(context.Background(), &event)

		// No traceparent should be set
		assert.Nil(t, event.Extensions()["traceparent"])
	})
}

func TestRoundTrip(t *testing.T) {
	// Test inject -> extract -> verify parent-child relationship
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Create a parent span
	ctx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")

	// Inject into CloudEvent
	event := cloudevents.NewEvent()
	InjectTraceIntoCloudEvent(ctx, &event)
	parentSpan.End()

	// Extract from CloudEvent
	extractedCtx := SpanFromCloudEvent(context.Background(), event)

	// Create a child span from extracted context
	_, childSpan := tp.Tracer("test").Start(extractedCtx, "child")
	defer childSpan.End()

	// Child span should have same trace ID as parent
	assert.Equal(t, parentSpan.SpanContext().TraceID(), childSpan.SpanContext().TraceID())
	// But different span IDs
	assert.NotEqual(t, parentSpan.SpanContext().SpanID(), childSpan.SpanContext().SpanID())
}

func setupTestTracer(t *testing.T) (trace.Tracer, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return tp.Tracer("test-span-kind"), exporter
}

func TestStartServerSpan(t *testing.T) {
	tracer, exporter := setupTestTracer(t)

	ctx, span := StartServerSpan(context.Background(), tracer, "server-op")
	require.NotNil(t, ctx)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, trace.SpanKindServer, spans[0].SpanKind)
	assert.Equal(t, "server-op", spans[0].Name)
}

func TestStartProducerSpan(t *testing.T) {
	tracer, exporter := setupTestTracer(t)

	ctx, span := StartProducerSpan(context.Background(), tracer, "producer-op")
	require.NotNil(t, ctx)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, trace.SpanKindProducer, spans[0].SpanKind)
	assert.Equal(t, "producer-op", spans[0].Name)
}

func TestStartConsumerSpan(t *testing.T) {
	tracer, exporter := setupTestTracer(t)

	ctx, span := StartConsumerSpan(context.Background(), tracer, "consumer-op")
	require.NotNil(t, ctx)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, trace.SpanKindConsumer, spans[0].SpanKind)
	assert.Equal(t, "consumer-op", spans[0].Name)
}

func TestStartClientSpan(t *testing.T) {
	tracer, exporter := setupTestTracer(t)

	ctx, span := StartClientSpan(context.Background(), tracer, "client-op")
	require.NotNil(t, ctx)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, trace.SpanKindClient, spans[0].SpanKind)
	assert.Equal(t, "client-op", spans[0].Name)
}
