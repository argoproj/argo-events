package common

import (
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithID(t *testing.T) {
	e := event.New()
	err := WithID("custom-id")(&e)
	require.NoError(t, err)
	assert.Equal(t, "custom-id", e.ID())
}

func TestWithSource(t *testing.T) {
	e := event.New()
	err := WithSource("https://example.com/source")(&e)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/source", e.Source())
}

func TestWithType(t *testing.T) {
	e := event.New()
	err := WithType("com.example.test")(&e)
	require.NoError(t, err)
	assert.Equal(t, "com.example.test", e.Type())
}

func TestWithSubject(t *testing.T) {
	e := event.New()
	err := WithSubject("my-subject")(&e)
	require.NoError(t, err)
	assert.Equal(t, "my-subject", e.Subject())
}

func TestWithTime(t *testing.T) {
	e := event.New()
	ts := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	err := WithTime(ts)(&e)
	require.NoError(t, err)
	assert.Equal(t, ts, e.Time())
}

func TestWithExtension(t *testing.T) {
	e := event.New()
	err := WithExtension("traceparent", "00-abc-def-01")(&e)
	require.NoError(t, err)
	assert.Equal(t, "00-abc-def-01", e.Extensions()["traceparent"])
}

func TestWithCloudEvent(t *testing.T) {
	t.Run("overrides all non-zero attributes", func(t *testing.T) {
		incoming := cloudevents.NewEvent()
		incoming.SetID("incoming-id")
		incoming.SetSource("https://external.com/source")
		incoming.SetType("com.external.event")
		incoming.SetSubject("external-subject")
		ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
		incoming.SetTime(ts)
		incoming.SetDataContentType("application/xml")
		incoming.SetExtension("traceparent", "00-trace-span-01")
		incoming.SetExtension("customext", "value123")

		// Create a default event (simulating what eventing.go does)
		e := event.New()
		e.SetID("generated-uuid")
		e.SetSource("my-eventsource")
		e.SetType("WebhookEvent")
		e.SetSubject("my-event")
		e.SetTime(time.Now())

		err := WithCloudEvent(incoming)(&e)
		require.NoError(t, err)

		assert.Equal(t, "incoming-id", e.ID())
		assert.Equal(t, "https://external.com/source", e.Source())
		assert.Equal(t, "com.external.event", e.Type())
		assert.Equal(t, "external-subject", e.Subject())
		assert.Equal(t, ts, e.Time())
		assert.Equal(t, "application/xml", e.DataContentType())
		assert.Equal(t, "00-trace-span-01", e.Extensions()["traceparent"])
		assert.Equal(t, "value123", e.Extensions()["customext"])
	})

	t.Run("preserves defaults for empty incoming attributes", func(t *testing.T) {
		incoming := cloudevents.NewEvent()
		// Only set some attributes, leave others empty
		incoming.SetSource("https://external.com/source")
		incoming.SetExtension("myext", "val")

		e := event.New()
		e.SetID("generated-uuid")
		e.SetType("WebhookEvent")
		e.SetSubject("my-event")
		defaultTime := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)
		e.SetTime(defaultTime)

		err := WithCloudEvent(incoming)(&e)
		require.NoError(t, err)

		// Overridden by incoming
		assert.Equal(t, "https://external.com/source", e.Source())
		assert.Equal(t, "val", e.Extensions()["myext"])

		// Preserved from defaults (incoming was empty)
		assert.Equal(t, "generated-uuid", e.ID())
		assert.Equal(t, "WebhookEvent", e.Type())
		assert.Equal(t, "my-event", e.Subject())
		assert.Equal(t, defaultTime, e.Time())
	})

	t.Run("no extensions on incoming does not clear existing", func(t *testing.T) {
		incoming := cloudevents.NewEvent()
		incoming.SetID("new-id")

		e := event.New()
		e.SetID("old-id")
		e.SetExtension("existing", "keep-me")

		err := WithCloudEvent(incoming)(&e)
		require.NoError(t, err)

		assert.Equal(t, "new-id", e.ID())
		assert.Equal(t, "keep-me", e.Extensions()["existing"])
	})
}
