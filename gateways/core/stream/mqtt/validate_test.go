package mqtt

import (
	"context"
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
	es := &gateways.EventSource{
		Data: &configValue,
	}
	ctx := context.Background()
	_, err := ce.ValidateEventSource(ctx, es)
	assert.Nil(t, err)

	configValue = `
url: tcp://mqtt.argo-events:1883
topic: foo
`
	es.Data = &configValue
	_, err = ce.ValidateEventSource(ctx, es)
	assert.NotNil(t, err)
}
