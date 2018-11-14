package kafka

import (
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
	ctx := &gateways.ConfigContext{Data: &gateways.ConfigData{}}
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)

	configValue = `
topic: foo
`
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}
