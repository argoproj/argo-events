package nats

import (
	"testing"
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
)

var (
	configKey = "testConfig"
	configValue = `
url: nats://nats.argo-events:4222
subject: foo
`
)

func TestNatsConfigExecutor_Validate(t *testing.T) {
	ce := &NatsConfigExecutor{}
	ctx := &gateways.ConfigContext{}
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
