package eventsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	t.Run("validate calendar eventsource", func(t *testing.T) {
		testEventSource := fakeEmptyEventSource()
		testEventSource.Spec.Calendar = fakeCalendarEventSourceMap("test")
		err := ValidateEventSource(testEventSource)
		assert.NoError(t, err)
	})

	t.Run("validate good mixed types eventsource", func(t *testing.T) {
		testEventSource := fakeEmptyEventSource()
		testEventSource.Spec.Kafka = fakeKafkaEventSourceMap("test1")
		testEventSource.Spec.MQTT = fakeMQTTEventSourceMap("test2")
		err := ValidateEventSource(testEventSource)
		assert.NoError(t, err)
	})

	t.Run("validate bad mixed types eventsource", func(t *testing.T) {
		testEventSource := fakeEmptyEventSource()
		testEventSource.Spec.Kafka = fakeKafkaEventSourceMap("test1")
		testEventSource.Spec.Webhook = fakeWebhookEventSourceMap("test2")
		err := ValidateEventSource(testEventSource)
		assert.Error(t, err)
		assert.Equal(t, "event sources with rolling update and recreate update strategy can not be put together", err.Error())
	})

	t.Run("validate bad mixed types eventsource - duplicated name", func(t *testing.T) {
		testEventSource := fakeEmptyEventSource()
		testEventSource.Spec.Kafka = fakeKafkaEventSourceMap("test")
		testEventSource.Spec.MQTT = fakeMQTTEventSourceMap("test")
		err := ValidateEventSource(testEventSource)
		assert.Error(t, err)
		assert.Equal(t, "more than one \"test\" found in the spec", err.Error())
	})
}
