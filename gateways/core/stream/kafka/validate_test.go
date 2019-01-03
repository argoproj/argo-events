package kafka

import (
	"context"
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
url: kafka.argo-events:9092
topic: foo
partition: "0"
`
)

func TestKafkaConfigExecutor_Validate(t *testing.T) {
	ce := &KafkaConfigExecutor{}
	es := &gateways.EventSource{Data: &configValue}
	ctx := context.Background()
	_, err := ce.ValidateEventSource(ctx, es)
	assert.Nil(t, err)

	configValue = `
topic: foo
`
	es.Data = &configValue
	_, err = ce.ValidateEventSource(ctx, es)
	assert.NotNil(t, err)
}
