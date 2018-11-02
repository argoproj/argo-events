package calendar

import (
	"testing"
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
)

var (
	configKey = "testConfig"
	configValue = `
schedule: 30 * * * *
`
)

func getConfigContext() *gateways.ConfigContext {
	return &gateways.ConfigContext{
		StopCh: make(chan struct{}),
		Data: &gateways.ConfigData{
			Src: configKey,
		},
	}
}

func testConfigs(t *testing.T, config string) error {
	i, err := gateways.ParseGatewayConfig(config)
	assert.Nil(t, err)
	ce := &CalendarConfigExecutor{}
	ctx := getConfigContext()
	ctx.Data.Config = i
	return ce.Validate(ctx)

}

func TestCalendarConfigExecutor_Validate(t *testing.T) {
	err := testConfigs(t, configValue)
	assert.Nil(t, err)
	configValue = `
interval: 55s`
	err = testConfigs(t, configValue)
	assert.Nil(t, err)
	configValue = ""
	err = testConfigs(t, configValue)
	assert.NotNil(t, err)
}

