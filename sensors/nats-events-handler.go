package sensors

import (
	"strconv"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/nats-io/go-nats"
	snats "github.com/nats-io/go-nats-streaming"
)

// NatsEventProtocol handles events sent over NATS
func (sec *sensorExecutionCtx) NatsEventProtocol() {
	var err error

	switch sec.sensor.Spec.EventProtocol.Nats.Type {
	case v1alpha1.Standard:
		if sec.nconn.standard == nil {
			sec.nconn.standard, err = nats.Connect(sec.sensor.Spec.EventProtocol.Nats.URL)
			if err != nil {
				sec.log.Panic().Err(err).Msg("failed to connect to nats server")
			}
		}
		for _, dependency := range sec.sensor.Spec.Dependencies {
			if dependency.Connected {
				continue
			}
			if _, err := sec.getNatsStandardSubscription(dependency.Name); err != nil {
				sec.log.Panic().Err(err).Str("event-source-name", dependency.Name).Msg("failed to get the nats subscription")
			}
			dependency.Connected = true
		}

	case v1alpha1.Streaming:
		if sec.nconn.stream == nil {
			sec.nconn.stream, err = snats.Connect(sec.sensor.Spec.EventProtocol.Nats.ClusterId, sec.sensor.Spec.EventProtocol.Nats.ClientId, snats.NatsURL(sec.sensor.Spec.EventProtocol.Nats.URL))
			if err != nil {
				sec.log.Panic().Err(err).Msg("failed to connect to nats streaming server")
			}
		}
		for _, dependency := range sec.sensor.Spec.Dependencies {
			if dependency.Connected {
				continue
			}
			if _, err := sec.getNatsStreamingSubscription(dependency.Name); err != nil {
				sec.log.Panic().Err(err).Str("event-source-name", dependency.Name).Msg("failed to get the nats subscription")
			}
			dependency.Connected = true
		}
	}
}

// getNatsStandardSubscription returns a standard nats subscription
func (sec *sensorExecutionCtx) getNatsStandardSubscription(eventSource string) (*nats.Subscription, error) {
	return sec.nconn.standard.QueueSubscribe(eventSource, eventSource, func(msg *nats.Msg) {
		sec.processNatsMessage(msg.Data, eventSource)
	})
}

// getNatsStreamingSubscription returns a streaming nats subscription
func (sec *sensorExecutionCtx) getNatsStreamingSubscription(eventSource string) (snats.Subscription, error) {
	if sec.sensor.Spec.EventProtocol.Nats.StartWithLastReceived {
		return sec.nconn.stream.Subscribe(eventSource, func(msg *snats.Msg) {
			sec.processNatsMessage(msg.Data, eventSource)
		}, snats.StartWithLastReceived())
	}

	if sec.sensor.Spec.EventProtocol.Nats.DeliverAllAvailable {
		return sec.nconn.stream.Subscribe(eventSource, func(msg *snats.Msg) {
			sec.processNatsMessage(msg.Data, eventSource)
		}, snats.DeliverAllAvailable())
	}

	if sec.sensor.Spec.EventProtocol.Nats.Durable {
		return sec.nconn.stream.Subscribe(eventSource, func(msg *snats.Msg) {
			sec.processNatsMessage(msg.Data, eventSource)
		}, snats.DurableName(eventSource))
	}

	if sec.sensor.Spec.EventProtocol.Nats.StartAtSequence != "" {
		sequence, err := strconv.Atoi(sec.sensor.Spec.EventProtocol.Nats.StartAtSequence)
		if err != nil {
			return nil, err
		}
		return sec.nconn.stream.Subscribe(eventSource, func(msg *snats.Msg) {
			sec.processNatsMessage(msg.Data, eventSource)
		}, snats.StartAtSequence(uint64(sequence)))
	}

	if sec.sensor.Spec.EventProtocol.Nats.StartAtTime != "" {
		startTime, err := time.Parse(common.StandardTimeFormat, sec.sensor.Spec.EventProtocol.Nats.StartAtTime)
		if err != nil {
			return nil, err
		}
		return sec.nconn.stream.Subscribe(eventSource, func(msg *snats.Msg) {
			sec.processNatsMessage(msg.Data, eventSource)
		}, snats.StartAtTime(startTime))
	}

	if sec.sensor.Spec.EventProtocol.Nats.StartAtTimeDelta != "" {
		duration, err := time.ParseDuration(sec.sensor.Spec.EventProtocol.Nats.StartAtTimeDelta)
		if err != nil {
			return nil, err
		}
		return sec.nconn.stream.Subscribe(eventSource, func(msg *snats.Msg) {
			sec.processNatsMessage(msg.Data, eventSource)
		}, snats.StartAtTimeDelta(duration))
	}

	return sec.nconn.stream.Subscribe(eventSource, func(msg *snats.Msg) {
		sec.processNatsMessage(msg.Data, eventSource)
	})
}

// processNatsMessage handles a nats message payload
func (sec *sensorExecutionCtx) processNatsMessage(msg []byte, eventSource string) {
	event, err := sec.parseEvent(msg)
	if err != nil {
		sec.log.Error().Err(err).Str("event-source-name", eventSource).Msg("failed to parse message into event")
		return
	}
	// validate whether the event is from gateway that this sensor is watching and send event over internal queue if valid
	if sec.sendEventToInternalQueue(event, nil) {
		sec.log.Info().Str("event-source-name", eventSource).Msg("event successfully sent over internal queue")
		return
	}
	sec.log.Warn().Str("event-source-name", eventSource).Msg("event is from unknown source")
}
