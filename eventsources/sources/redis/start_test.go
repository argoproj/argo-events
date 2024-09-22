package redis

import (
	"encoding/json"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/events"
	metrics "github.com/argoproj/argo-events/metrics"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func Test_HandleOne(t *testing.T) {
	el := &EventListener{
		EventSourceName:  "esName",
		EventName:        "eName",
		RedisEventSource: v1alpha1.RedisEventSource{},
		Metrics:          metrics.NewMetrics("ns"),
	}

	msg := &redis.Message{
		Channel: "ch",
		Pattern: "p",
		Payload: `{"a": "b"}`,
	}

	getDispatcher := func(isJson bool) func(d []byte, opts ...eventsourcecommon.Option) error {
		return func(d []byte, opts ...eventsourcecommon.Option) error {
			eventData := &events.RedisEventData{}
			err := json.Unmarshal(d, eventData)
			assert.NoError(t, err)
			assert.Equal(t, msg.Pattern, eventData.Pattern)
			assert.Equal(t, msg.Channel, eventData.Channel)
			if !isJson {
				s, ok := eventData.Body.(string)
				assert.True(t, ok)
				assert.Equal(t, msg.Payload, s)
			} else {
				s, ok := eventData.Body.(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "b", s["a"])
			}
			return nil
		}
	}

	err := el.handleOne(msg, getDispatcher(el.RedisEventSource.JSONBody), zaptest.NewLogger(t).Sugar())
	assert.NoError(t, err)

	el.RedisEventSource.JSONBody = true
	err = el.handleOne(msg, getDispatcher(el.RedisEventSource.JSONBody), zaptest.NewLogger(t).Sugar())
	assert.NoError(t, err)
}
