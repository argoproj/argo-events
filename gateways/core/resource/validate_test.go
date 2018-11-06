package resource

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
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

func TestResourceConfigExecutor_Validate(t *testing.T) {
	ce := &ResourceConfigExecutor{}
	ctx := &gateways.ConfigContext{
		Data: &gateways.ConfigData{},
	}
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)
	configValue = `
group: "argoproj.io"
version: "v1alpha1"
filter:
   labels:
   workflows.argoproj.io/phase: Succeeded
   name: "my-workflow"
`
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
	configValue = ""
	ctx.Data.Config = configValue
	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}
