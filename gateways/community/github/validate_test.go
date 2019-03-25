package github

import (
	"context"
	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

var (
	configKey  = "testConfig"
	configId   = "1234"
	goodConfig = `
id: 1234
hook:
 endpoint: "/push"
 port: "12000"
 url: "http://webhook-gateway-svc"
owner: "asd"
repository: "dsa"
events:
- PushEvents
apiToken:
  key: accesskey
  name: githab-access
`

	badConfig = `
events:
- PushEvents
accessToken:
  key: accesskey
  name: gitlab-access
`
)

func TestValidateGithubEventSource(t *testing.T) {
	convey.Convey("Given a valid github event source spec, parse it and make sure no error occurs", t, func() {
		ese := &GithubEventSourceExecutor{}
		valid, _ := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Name: configKey,
			Id:   configId,
			Data: goodConfig,
		})
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(valid.IsValid, convey.ShouldBeTrue)
	})

	convey.Convey("Given an invalid github event source spec, parse it and make sure error occurs", t, func() {
		ese := &GithubEventSourceExecutor{}
		valid, _ := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Data: badConfig,
			Id:   configId,
			Name: configKey,
		})
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(valid.IsValid, convey.ShouldBeFalse)
		convey.So(valid.Reason, convey.ShouldNotBeEmpty)
	})
}
