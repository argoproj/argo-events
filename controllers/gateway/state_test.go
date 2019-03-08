package gateway

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/smartystreets/goconvey/convey"
)

func TestPersistUpdates(t *testing.T) {
	convey.Convey("Given a gateway resource", t, func() {
		namespace := "argo-events"
		client := fake.NewSimpleClientset()
		logger := common.GetLoggerContext(common.LoggerConf()).Logger()
		gw, err := getGateway()
		convey.So(err, convey.ShouldBeNil)

		convey.Convey("Create the gateway", func() {
			gw, err = client.ArgoprojV1alpha1().Gateways(namespace).Create(gw)
			convey.So(err, convey.ShouldBeNil)
			convey.So(gw, convey.ShouldNotBeNil)

			gw.ObjectMeta.Labels = map[string]string{
				"default": "default",
			}

			convey.Convey("Update the gateway", func() {
				updatedGw, err := PersistUpdates(client, gw, &logger)
				convey.So(err, convey.ShouldBeNil)
				convey.So(updatedGw, convey.ShouldNotEqual, gw)
				convey.So(updatedGw.Labels, convey.ShouldResemble, gw.Labels)

				updatedGw.Labels["new"] = "new"

				convey.Convey("Reapply the gateway", func() {
					err := ReapplyUpdate(client, updatedGw)
					convey.So(err, convey.ShouldBeNil)
					convey.So(len(updatedGw.Labels), convey.ShouldEqual, 2)
				})
			})
		})
	})
}
