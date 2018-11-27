package gitlab

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	configKey   = "testConfig"
	configValue = `
projectId: "28"
url: "http://webhook-gateway-gateway-svc/push"
event: "PushEvents"
accessToken:
    key: accesskey
    name: gitlab-access
enableSSLVerification: false   
gitlabBaseUrl: "http://gitlab.com/"
`
)

func TestGitlabExecutor_Validate(t *testing.T) {
	ce := &GitlabExecutor{}
	ctx := &gateways.ConfigContext{
		Data: &gateways.ConfigData{},
	}
	ctx.Data.Config = configValue
	err := ce.Validate(ctx)
	assert.Nil(t, err)

	badConfig := `
url: "http://webhook-gateway-gateway-svc/push"
event: "PushEvents"
accessToken:
    key: accesskey
    name: gitlab-access
`

	ctx.Data.Config = badConfig

	err = ce.Validate(ctx)
	assert.NotNil(t, err)
}
