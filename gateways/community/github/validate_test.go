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

func TestValidateGithubEventSource(t *testing.T) {
	convey.Convey("Given a valid github event source spec, parse it and make sure no error occurs", t, func() {
		ese := &GithubEventSourceExecutor{}
		valid, err := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Name: configKey,
			Id:   configId,
			Data: goodConfig,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(valid.IsValid, convey.ShouldBeTrue)
	})

	convey.Convey("Given an invalid github event source spec, parse it and make sure error occurs", t, func() {
		ese := &GithubEventSourceExecutor{}
		valid, err := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
			Data: badConfig,
			Id:   configId,
			Name: configKey,
		})
		convey.So(err, convey.ShouldNotBeNil)
		convey.So(valid, convey.ShouldNotBeNil)
		convey.So(valid.IsValid, convey.ShouldBeFalse)
		convey.So(valid.Reason, convey.ShouldNotBeEmpty)
	})
}
