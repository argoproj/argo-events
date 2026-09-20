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

package amqp

import (
	"errors"
	"testing"

	amqplib "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
)

// mockAcknowledger records the Ack/Nack calls made against a Delivery so
// tests can assert on manual acknowledgement behavior.
type mockAcknowledger struct {
	acked   bool
	nacked  bool
	requeue bool
}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	m.acked = true
	return nil
}

func (m *mockAcknowledger) Nack(tag uint64, multiple, requeue bool) error {
	m.nacked = true
	m.requeue = requeue
	return nil
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	return nil
}

func newEventListener() *EventListener {
	return &EventListener{
		EventSourceName: "esName",
		EventName:       "eName",
		Metrics:         metrics.NewMetrics("ns"),
	}
}

func TestHandleOne_ManualAck(t *testing.T) {
	el := newEventListener()
	amqpEventSource := &aev1.AMQPEventSource{
		Consume: &aev1.AMQPConsumeConfig{AutoAck: false},
	}
	ack := &mockAcknowledger{}
	msg := amqplib.Delivery{Acknowledger: ack, Body: []byte(`{"a":"b"}`)}

	dispatch := func(b []byte, opts ...eventsourcecommon.Option) error { return nil }
	err := el.handleOne(amqpEventSource, msg, dispatch, zaptest.NewLogger(t).Sugar())
	assert.NoError(t, err)
	assert.True(t, ack.acked, "expected the message to be acked after a successful dispatch")
	assert.False(t, ack.nacked)
}

func TestHandleOne_ManualNackOnDispatchFailure(t *testing.T) {
	el := newEventListener()
	amqpEventSource := &aev1.AMQPEventSource{
		Consume: &aev1.AMQPConsumeConfig{AutoAck: false},
	}
	ack := &mockAcknowledger{}
	msg := amqplib.Delivery{Acknowledger: ack, Body: []byte(`{"a":"b"}`)}

	dispatch := func(b []byte, opts ...eventsourcecommon.Option) error { return errors.New("dispatch failed") }
	err := el.handleOne(amqpEventSource, msg, dispatch, zaptest.NewLogger(t).Sugar())
	assert.Error(t, err)
	assert.True(t, ack.nacked, "expected the message to be nacked and requeued after a failed dispatch")
	assert.True(t, ack.requeue)
	assert.False(t, ack.acked)
}

func TestHandleOne_AutoAckSkipsManualAck(t *testing.T) {
	el := newEventListener()
	amqpEventSource := &aev1.AMQPEventSource{
		Consume: &aev1.AMQPConsumeConfig{AutoAck: true},
	}
	// No Acknowledger set: manual Ack/Nack would return
	// amqplib.ErrDeliveryNotInitialized if mistakenly called.
	msg := amqplib.Delivery{Body: []byte(`{"a":"b"}`)}

	dispatch := func(b []byte, opts ...eventsourcecommon.Option) error { return nil }
	err := el.handleOne(amqpEventSource, msg, dispatch, zaptest.NewLogger(t).Sugar())
	assert.NoError(t, err)
}

func TestParseYamlTable(t *testing.T) {
	table, err := parseYamlTable("")
	assert.Nil(t, err)
	assert.Nil(t, table)
	table, err = parseYamlTable(`:noKey`)
	assert.NotNil(t, err)
	assert.Nil(t, table)
	table, err = parseYamlTable("x-queue-type: quorum")
	assert.Nil(t, err)
	assert.NotNil(t, table)
	assert.True(t, len(table) == 1)
	table, err = parseYamlTable("x-expires: 86400000")
	assert.Nil(t, err)
	assert.NotNil(t, table)
	val, ok := table["x-expires"]
	assert.True(t, ok)
	switch n := val.(type) {
	case int:
		assert.Equal(t, int(86400000), n)
	case int64:
		assert.Equal(t, int64(86400000), n)
	case uint64:
		assert.Equal(t, uint64(86400000), n)
	default:
		assert.Failf(t, "expected integer YAML scalar", "got %T (%v)", val, val)
	}
	table, err = parseYamlTable("key-one: thing1\nkey-two: thing2")
	assert.Nil(t, err)
	assert.NotNil(t, table)
	assert.True(t, len(table) == 2)
	assert.Equal(t, "thing1", table["key-one"].(string))
	assert.Equal(t, "thing2", table["key-two"].(string))
}
