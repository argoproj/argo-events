package sensor

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	fakesensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	"github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSensorState(t *testing.T) {
	fakeSensorClient := fakesensor.NewSimpleClientset()
	logger := common.GetLoggerContext(common.LoggerConf()).Logger()
	sn := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-sensor",
			Namespace: "test",
		},
	}

	convey.Convey("Given a sensor", t, func() {
		convey.Convey("Create the sensor", func() {
			sn, err := fakeSensorClient.ArgoprojV1alpha1().Sensors(sn.Namespace).Create(sn)
			convey.So(err, convey.ShouldBeNil)
			convey.So(sn, convey.ShouldNotBeNil)
		})

		convey.Convey("Initialize a new node", func() {
			status := InitializeNode(sn, "first_node", v1alpha1.NodeTypeEventDependency, &logger)
			convey.So(status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseNew)
		})

		convey.Convey("Persist updates to sn", func() {
			sensor, err := PersistUpdates(fakeSensorClient, sn, "1", &logger)
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(sensor.Status.Nodes), convey.ShouldEqual, 1)
		})

		convey.Convey("Mark sn node state to active", func() {
			status := MarkNodePhase(sn, "first_node", v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, &v1alpha1.Event{
				Payload: []byte("test payload"),
			}, &logger)
			convey.So(status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseActive)
		})

		convey.Convey("Reapply sn update", func() {
			err := ReapplyUpdate(fakeSensorClient, sn)
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("Fetch sn and check updates are applied", func() {
			sensor, err := fakeSensorClient.ArgoprojV1alpha1().Sensors(sn.Namespace).Get(sn.Name, metav1.GetOptions{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(len(sensor.Status.Nodes), convey.ShouldEqual, 1)
			convey.Convey("Get the first_node node", func() {
				node := GetNodeByName(sensor, "first_node")
				convey.So(node, convey.ShouldNotBeNil)
				convey.So(node.Phase, convey.ShouldEqual, v1alpha1.NodePhaseActive)
			})
		})
	})
}
