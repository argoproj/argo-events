package amqp

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAMQPConfigExecutor_StopConfig(t *testing.T) {
	ce := &AMQPConfigExecutor{}
	ctx := &gateways.ConfigContext{
		StopChan: make(chan struct{}),
	}
	ctx.Active = true
	go func() {
		msg := <-ctx.StopChan
		assert.Equal(t, msg, struct{}{})
	}()
	ce.StopConfig(ctx)
	assert.Equal(t, false, ctx.Active)
}
