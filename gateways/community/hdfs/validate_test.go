package hdfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateEventSource(t *testing.T) {
	convey.Convey("Given a hdfs event source spec, parse it and make sure no error occurs", t, func() {
		ese := &EventSourceExecutor{}
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", gwcommon.EventSourceDir, "hdfs.yaml"))
		convey.So(err, convey.ShouldBeNil)

		var cm *corev1.ConfigMap
		err = yaml.Unmarshal(content, &cm)
		convey.So(err, convey.ShouldBeNil)
		convey.So(cm, convey.ShouldNotBeNil)

		err = common.CheckEventSourceVersion(cm)
		convey.So(err, convey.ShouldBeNil)

		for key, value := range cm.Data {
			valid, _ := ese.ValidateEventSource(context.Background(), &gateways.EventSource{
				Name:    key,
				Id:      common.Hasher(key),
				Data:    value,
				Version: cm.Labels[common.LabelArgoEventsEventSourceVersion],
			})
			convey.So(valid, convey.ShouldNotBeNil)
			convey.So(valid.IsValid, convey.ShouldBeTrue)
		}
	})
}
