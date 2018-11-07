package storagegrid


import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
endpoint: "/"
port: "8080"
events:
    - "ObjectCreated:Put"
filter:
    suffix: ".txt"
    prefix: "hello-"
`
)

func TestStorageGridConfigExecutor_Validate(t *testing.T) {
	s3Config := &StorageGridConfigExecutor{}
	ctx := &gateways.ConfigContext{
		Data: &gateways.ConfigData{},
	}
	ctx.Data.Config = configValue
	err := s3Config.Validate(ctx)
	assert.Nil(t, err)

	badConfig := `
events:
    - "ObjectCreated:Put"
filter:
    suffix: ".txt"
    prefix: "hello-"
`

	ctx.Data.Config = badConfig

	err = s3Config.Validate(ctx)
	assert.NotNil(t, err)
}
