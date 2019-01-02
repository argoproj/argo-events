package webhook

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
endpoint: "/bar"
method: "POST"
`
)

func TestNatsConfigExecutor_Validate(t *testing.T) {
	ce := &WebhookConfigExecutor{}
	ctx := &gateways.EventSourceContext{
		Data: &gateways.EventSourceData{},
	}
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)

	configValue = `
`
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}
