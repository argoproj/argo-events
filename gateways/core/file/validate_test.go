package file

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
directory: "/bin/"
type: CREATE
path: x.txt
`
)

func TestFileWatcherConfigExecutor_Validate(t *testing.T) {
	ce := &FileWatcherConfigExecutor{}
	ctx := &gateways.ConfigContext{
		Data: &gateways.ConfigData{},
	}
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)
	configValue = `
directory: "/bin/"
type: CREATE
`
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
	configValue = ``
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}
