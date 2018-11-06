package mqtt

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
url: tcp://mqtt.argo-events:1883
topic: foo
clientId: 1
`
)

func TestMqttConfigExecutor_Validate(t *testing.T) {
	ce := &MqttConfigExecutor{}
	ctx := gateways.GetDefaultConfigContext(configKey)
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)

	configValue = `
url: tcp://mqtt.argo-events:1883
topic: foo
`
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}
