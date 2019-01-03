package webhook

import (
	"context"
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
endpoint: "/bar"
port: "10000"
method: "POST"
`
)

func TestNatsConfigExecutor_Validate(t *testing.T) {
	ce := &WebhookConfigExecutor{}
	es := &gateways.EventSource{
		Data: &configValue,
	}
	ctx := context.Background()
	_, err := ce.ValidateEventSource(ctx, es)
	assert.Nil(t, err)

	configValue = `
`
	es.Data = &configValue
	_, err = ce.ValidateEventSource(ctx, es)
	assert.NotNil(t, err)
}
