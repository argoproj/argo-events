package amqp

import (
	"context"
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
	ce := &AMQPEventSourceExecutor{}
	es := &gateways.EventSource{
		Data: &configValue,
	}
	ctx := context.Background()
	_, err := ce.ValidateEventSource(ctx, es)
	assert.Nil(t, err)

	configValue = `
url: amqp://amqp.argo-events:5672
exchangeName: fooExchangeName
exchangeType: fanout
`
	es.Data = &configValue
	_, err = ce.ValidateEventSource(ctx, es)
	assert.NotNil(t, err)
}
