/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	ce := &GitlabEventSourceExecutor{}
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
