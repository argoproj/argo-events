package common

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
	"net"
	"testing"
)

func getFakeRouteConfig() *RouteConfig {
	return &RouteConfig{
		Webhook: &Webhook{
			Endpoint: "/fake",
			Port:     "12000",
			URL:      "test-url",
		},
		EventSource: &gateways.EventSource{
			Name: "fake-event-source",
			Data: "hello",
			Id:   "123",
		},
		Log:     common.GetLoggerContext(common.LoggerConf()).Logger(),
		Configs: make(map[string]interface{}),
		StartCh: make(chan struct{}),
	}
}

func TestInitRouteChannels(t *testing.T) {
	convey.Convey("Given a webhook helper, initialize the routes channels", t, func() {
		webhookHelper := NewWebhookHelper()
		go InitRouteChannels(webhookHelper)

		rc := getFakeRouteConfig()

		webhookHelper.RouteActivateChan <- rc

		<-rc.StartCh

		convey.Convey("Confirm the server is running on specified port", func() {
			conn, err := net.Dial("tcp", ":12000")
			convey.So(err, convey.ShouldBeNil)
			convey.So(conn, convey.ShouldNotBeNil)
		})
	})

}
