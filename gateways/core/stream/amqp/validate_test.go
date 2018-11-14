package amqp

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
url: amqp://amqp.argo-events:5672
exchangeName: fooExchangeName
exchangeType: fanout
routingKey: fooRoutingKey
`
)

func TestAMQPConfigExecutor_Validate(t *testing.T) {
	ce := &AMQPConfigExecutor{}
	ctx := &gateways.ConfigContext{
		Data: &gateways.ConfigData{},
	}
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)

	configValue = `
url: amqp://amqp.argo-events:5672
exchangeName: fooExchangeName
exchangeType: fanout
`
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}
