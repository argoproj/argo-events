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

package gateways

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/smartystreets/goconvey/convey"
)

func TestGatewayState(t *testing.T) {
	gc := getGatewayConfig()
	convey.Convey("Given a gateway", t, func() {
		convey.Convey("Create the gateway", func() {
			var err error
			gc.gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gc.gw.Namespace).Create(gc.gw)
			convey.So(err, convey.ShouldBeNil)

		})

		convey.Convey("Update gateway resource test-node node state to running", func() {
			gc.UpdateGatewayResourceState(&EventSourceStatus{
				Phase:   v1alpha1.NodePhaseRunning,
				Name:    "test-node",
				Message: "node is marked as running",
				Id:      "test-node",
			})
			convey.So(len(gc.gw.Status.Nodes), convey.ShouldEqual, 1)
			convey.So(gc.gw.Status.Nodes["test-node"].Phase, convey.ShouldEqual, v1alpha1.NodePhaseRunning)
		})

		updatedGw := gc.gw
		updatedGw.Spec.Watchers = &v1alpha1.NotificationWatchers{
			Sensors: []v1alpha1.SensorNotificationWatcher{
				{
					Name: "sensor-1",
				},
			},
		}

		convey.Convey("Update gateway watchers", func() {
			gc.UpdateGatewayResourceState(&EventSourceStatus{
				Phase:   v1alpha1.NodePhaseResourceUpdate,
				Name:    "test-node",
				Message: "gateway resource is updated",
				Id:      "test-node",
				Gw:      updatedGw,
			})
			convey.So(len(gc.gw.Spec.Watchers.Sensors), convey.ShouldEqual, 1)
		})

		convey.Convey("Update gateway resource test-node node state to completed", func() {
			gc.UpdateGatewayResourceState(&EventSourceStatus{
				Phase:   v1alpha1.NodePhaseCompleted,
				Name:    "test-node",
				Message: "node is marked completed",
				Id:      "test-node",
			})
			convey.So(gc.gw.Status.Nodes["test-node"].Phase, convey.ShouldEqual, v1alpha1.NodePhaseCompleted)
		})

		convey.Convey("Remove gateway resource test-node node", func() {
			gc.UpdateGatewayResourceState(&EventSourceStatus{
				Phase:   v1alpha1.NodePhaseRemove,
				Name:    "test-node",
				Message: "node is removed",
				Id:      "test-node",
			})
			convey.So(len(gc.gw.Status.Nodes), convey.ShouldEqual, 0)
		})
	})
}
