package hdfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
)

func TestValidateEventSource(t *testing.T) {
	convey.Convey("Given a hdfs event source spec, parse it and make sure no error occurs", t, func() {
		listener := &EventListener{}
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", gateways.EventSourceDir, "hdfs.yaml"))
		convey.So(err, convey.ShouldBeNil)

		var eventSource *v1alpha1.EventSource
		err = yaml.Unmarshal(content, &eventSource)
		convey.So(err, convey.ShouldBeNil)
		convey.So(eventSource, convey.ShouldNotBeNil)

		err = v1alpha1.ValidateEventSource(eventSource)
		convey.So(err, convey.ShouldBeNil)

		for key, value := range eventSource.Spec.HDFS {
			body, err := yaml.Marshal(value)
			convey.So(err, convey.ShouldBeNil)
			convey.So(err, convey.ShouldNotBeNil)

			valid, _ := listener.ValidateEventSource(context.Background(), &gateways.EventSource{
				Name:    key,
				Id:      common.Hasher(key),
				Value:   body,
				Version: eventSource.Spec.Version,
				Type:    string(eventSource.Spec.Type),
			})
			convey.So(valid, convey.ShouldNotBeNil)
			convey.So(valid.IsValid, convey.ShouldBeTrue)
		}
	})
}
