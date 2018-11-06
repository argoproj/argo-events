package calendar

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
schedule: 30 * * * *
`
)

func TestCalendarConfigExecutor_Validate(t *testing.T) {
	ce := CalendarConfigExecutor{}
	ctx := &gateways.ConfigContext{
		Data: &gateways.ConfigData{},
	}
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)
	configValue = `
interval: 55s`
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.Nil(t, err)
	configValue = ""
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}

