package common

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"
)

func getFakeRouteConfig() *RouteConfig {
	return &RouteConfig{
		Webhook: &Webhook{
			Endpoint: "/fake",
			Port:     "12000",
			URL:      "test-url",
			Method:   http.MethodGet,
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

func TestWebhook(t *testing.T) {
	convey.Convey("Given a webhook helper, initialize the routes channels", t, func() {
		webhookHelper := NewWebhookHelper()
		go InitRouteChannels(webhookHelper)

		rc := getFakeRouteConfig()

		webhookHelper.RouteActivateChan <- rc

		<-rc.StartCh

		convey.Convey("Confirm the server is running on specified port", func() {
			convey.So(len(webhookHelper.ActiveServers), convey.ShouldEqual, 1)

			conn, err := net.Dial("tcp", ":12000")
			convey.So(err, convey.ShouldBeNil)
			convey.So(conn, convey.ShouldNotBeNil)

			convey.Convey("Add route to server and validate that server serves the endpoint", func() {
				rc.RouteActiveHandler = func(writer http.ResponseWriter, request *http.Request, rc *RouteConfig) {
					if !webhookHelper.ActiveEndpoints[rc.Webhook.Endpoint].Active {
						common.SendErrorResponse(writer, "error")
						return
					}

					writer.WriteHeader(http.StatusOK)
					writer.Write([]byte("hello there"))
				}

				rc.activateRoute(webhookHelper)

				convey.So(len(webhookHelper.ActiveEndpoints), convey.ShouldEqual, 1)

				resp, err := http.Get("http://127.0.0.1:12000/fake")
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp, convey.ShouldNotBeNil)
				payload, err := ioutil.ReadAll(resp.Body)
				convey.So(err, convey.ShouldBeNil)
				convey.So(string(payload), convey.ShouldEqual, "hello there")
				convey.So(resp.Status, convey.ShouldEqual, "200 OK")

				convey.Convey("Deactivate an endpoint", func() {
					port, as := webhookHelper.ActiveServers[rc.Webhook.Port]
					convey.So(port, convey.ShouldNotBeEmpty)
					convey.So(as, convey.ShouldNotBeNil)

					webhookHelper.RouteDeactivateChan <- rc

					time.Sleep(time.Second * 2)

					resp, err := http.Get("http://127.0.0.1:12000/fake")
					convey.So(err, convey.ShouldBeNil)
					convey.So(resp, convey.ShouldNotBeNil)
					payload, err := ioutil.ReadAll(resp.Body)
					convey.So(err, convey.ShouldBeNil)
					convey.So(string(payload), convey.ShouldEqual, "error")
					convey.So(resp.Status, convey.ShouldEqual, "400 Bad Request")
				})
			})
		})
	})
}
