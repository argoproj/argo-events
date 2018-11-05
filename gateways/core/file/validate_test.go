package file

import (
	"testing"
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
)

var (
	configKey = "testConfig"
	configValue = `
directory: "/bin/"
type: CREATE
path: x.txt
`
)

func testConfig(t *testing.T, config string) error {
	i, err := (config)
	assert.Nil(t, err)
	ce := &FileWatcherConfigExecutor{}
	ctx := gateways.GetDefaultConfigContext(configKey)
	ctx.Data.Config = i
	return ce.Validate(ctx)
}

func TestFileWatcherConfigExecutor_Validate(t *testing.T) {
	err := testConfig(t, configValue)
	assert.Nil(t, err)
	configValue = `
directory: "/bin/"
type: CREATE
`
	err = testConfig(t, configValue)
	assert.NotNil(t, err)
	configValue = ``
	err = testConfig(t, configValue)
	assert.NotNil(t, err)
}
