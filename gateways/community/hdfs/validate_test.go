package hdfs

import (
	"context"
	"testing"

	"github.com/argoproj/argo-events/gateways"
	"github.com/smartystreets/goconvey/convey"
)

var (
	configKey  = "testConfig"
	configId   = "1234"
	goodConfig = `
directory: "/tmp/"
type: "CREATE"
path: x.txt
addresses:
- my-hdfs-namenode-0.my-hdfs-namenode.default.svc.cluster.local:8020
- my-hdfs-namenode-1.my-hdfs-namenode.default.svc.cluster.local:8020
hdfsUser: root
`

	badConfig = `
directory: "/tmp/"
type: "CREATE"
addresses:
- my-hdfs-namenode-0.my-hdfs-namenode.default.svc.cluster.local:8020
- my-hdfs-namenode-1.my-hdfs-namenode.default.svc.cluster.local:8020
hdfsUser: roots
`
)

func TestValidateEventSource(t *testing.T) {
	convey.Convey("Given a valid github event source spec, parse it and make sure no error occurs", t, func() {
		ese := &EventSourceExecutor{}
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
		ese := &EventSourceExecutor{}
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
