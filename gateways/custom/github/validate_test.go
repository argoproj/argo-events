package github

import (
	"context"
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	goodConfig = `
owner: "asd"
repository: "dsa"
url: "http://webhook-gateway-gateway-svc/push"
events:
- PushEvents
apiToken:
  key: accesskey
  name: githab-access
`

	badConfig = `
url: "http://webhook-gateway-gateway-svc/push"
events:
- PushEvents
accessToken:
  key: accesskey
  name: gitlab-access
`
)

func TestGithubExecutor_Validate(t *testing.T) {
	ce := &GithubEventSourceExecutor{}
	es := &gateways.EventSource{
		Data: &goodConfig,
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, err := ce.ValidateEventSource(ctx, es)
	assert.Nil(t, err)

	es.Data = &badConfig
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	_, err = ce.ValidateEventSource(ctx, es)
	assert.NotNil(t, err)
}
