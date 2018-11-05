package resource

import (
	"testing"
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
)

var (
	configKey = "testConfig"
	configValue = `
namespace: "argo-events"
group: "argoproj.io"
version: "v1alpha1"
kind: "Workflow"
filter:
    labels:
    workflows.argoproj.io/phase: Succeeded
    name: "my-workflow"
`
)

func testConfig(t *testing.T, config string) error {
	i, err := gateways.ParseGatewayConfig(config)
	assert.Nil(t, err)
	ce := &ResourceConfigExecutor{}
	ctx := gateways.GetDefaultConfigContext(configKey)
	ctx.Data.Config = i
	return ce.Validate(ctx)
}

func TestResourceConfigExecutor_Validate(t *testing.T) {
	err := testConfig(t, configValue)
	assert.Nil(t, err)
//	configValue = `
//group: "argoproj.io"
//version: "v1alpha1"
//filter:
//    labels:
//    workflows.argoproj.io/phase: Succeeded
//    name: "my-workflow"
//`
//	err = testConfig(t, configValue)
//	assert.NotNil(t, err)
//	configValue = ""
//	assert.NotNil(t, err)
}
