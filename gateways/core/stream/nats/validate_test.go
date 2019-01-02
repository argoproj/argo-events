package nats

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
url: nats://nats.argo-events:4222
subject: foo
`
)

func TestNatsConfigExecutor_Validate(t *testing.T) {
	ce := &NatsConfigExecutor{}
	ctx := &gateways.EventSourceContext{Data: &gateways.EventSourceData{}}
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)

	configValue = `
subject: foo
`
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}
