package webhook

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWebhookConfigExecutor_StopConfig(t *testing.T) {
	ce := &WebhookConfigExecutor{}
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
