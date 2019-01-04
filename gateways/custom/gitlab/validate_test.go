package gitlab

import (
	"context"
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
	es := &gateways.EventSource{
		Data: &configValue,
	}
	ctx := context.Background()
	_, err := ce.ValidateEventSource(ctx, es)
	assert.Nil(t, err)

	badConfig := `
url: "http://webhook-gateway-gateway-svc/push"
event: "PushEvents"
accessToken:
    key: accesskey
    name: gitlab-access
`

	es.Data = &badConfig
	_, err = ce.ValidateEventSource(ctx, es)
	assert.NotNil(t, err)
}
