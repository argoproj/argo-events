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

package main

import (
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGatewayState(t *testing.T) {
	gc := getGatewayContext()
	convey.Convey("Given a gateway", t, func() {
		convey.Convey("Create the gateway", func() {
			var err error
			gc.gateway, err = gc.gatewayClient.ArgoprojV1alpha1().Gateways(gc.gateway.Namespace).Create(gc.gateway)
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("Update gateway resource test-node node state to running", func() {
			gc.UpdateGatewayState(&EventSourceStatus{
				Phase:   v1alpha1.NodePhaseRunning,
				Name:    "test-node",
				Message: "node is marked as running",
				Id:      "test-node",
			})
			convey.So(len(gc.gateway.Status.Nodes), convey.ShouldEqual, 1)
			convey.So(gc.gateway.Status.Nodes["test-node"].Phase, convey.ShouldEqual, v1alpha1.NodePhaseRunning)
		})

		updatedGw := gc.gateway
		updatedGw.Spec.Watchers = &v1alpha1.NotificationWatchers{
			Sensors: []v1alpha1.SensorNotificationWatcher{
				{
					Name: "sensor-1",
				},
			},
		}

		convey.Convey("Update gateway watchers", func() {
			gc.UpdateGatewayState(&EventSourceStatus{
				Phase:   v1alpha1.NodePhaseResourceUpdate,
				Name:    "test-node",
				Message: "gateway resource is updated",
				Id:      "test-node",
				Gateway: updatedGw,
			})
			convey.So(len(gc.gateway.Spec.Watchers.Sensors), convey.ShouldEqual, 1)
		})

		convey.Convey("Update gateway resource test-node node state to completed", func() {
			gc.UpdateGatewayState(&EventSourceStatus{
				Phase:   v1alpha1.NodePhaseCompleted,
				Name:    "test-node",
				Message: "node is marked completed",
				Id:      "test-node",
			})
			convey.So(gc.gateway.Status.Nodes["test-node"].Phase, convey.ShouldEqual, v1alpha1.NodePhaseCompleted)
		})

		convey.Convey("Remove gateway resource test-node node", func() {
			gc.UpdateGatewayState(&EventSourceStatus{
				Phase:   v1alpha1.NodePhaseRemove,
				Name:    "test-node",
				Message: "node is removed",
				Id:      "test-node",
			})
			convey.So(len(gc.gateway.Status.Nodes), convey.ShouldEqual, 0)
		})
	})
}

func TestMarkGatewayNodePhase(t *testing.T) {
	convey.Convey("Given a node status, mark node state", t, func() {
		gc := getGatewayContext()
		nodeStatus := &EventSourceStatus{
			Name:    "fake",
			Id:      "1234",
			Message: "running",
			Phase:   v1alpha1.NodePhaseRunning,
			Gateway: gc.gateway,
		}
		gc.gateway.Status.Nodes = map[string]v1alpha1.NodeStatus{
			"1234": v1alpha1.NodeStatus{
				Phase:   v1alpha1.NodePhaseNew,
				Message: "init",
				Name:    "fake",
				ID:      "1234",
			},
		}

		resultStatus := gc.markGatewayNodePhase(nodeStatus)
		convey.So(resultStatus, convey.ShouldNotBeNil)
		convey.So(resultStatus.Name, convey.ShouldEqual, nodeStatus.Name)

		gc.gateway.Status.Nodes = map[string]v1alpha1.NodeStatus{
			"4567": v1alpha1.NodeStatus{
				Phase:   v1alpha1.NodePhaseNew,
				Message: "init",
				Name:    "fake",
				ID:      "1234",
			},
		}

		resultStatus = gc.markGatewayNodePhase(nodeStatus)
		convey.So(resultStatus, convey.ShouldBeNil)
	})
}

func TestGetNodeByID(t *testing.T) {
	convey.Convey("Given a node id, retrieve the node", t, func() {
		gc := getGatewayContext()
		gc.gateway.Status.Nodes = map[string]v1alpha1.NodeStatus{
			"1234": v1alpha1.NodeStatus{
				Phase:   v1alpha1.NodePhaseNew,
				Message: "init",
				Name:    "fake",
				ID:      "1234",
			},
		}
		status := gc.getNodeByID("1234")
		convey.So(status, convey.ShouldNotBeNil)
		convey.So(status.ID, convey.ShouldEqual, "1234")
	})
}

func TestInitializeNode(t *testing.T) {
	convey.Convey("Given a node, initialize it", t, func() {
		gc := getGatewayContext()
		status := gc.initializeNode("1234", "fake", "init")
		convey.So(status, convey.ShouldNotBeNil)
		convey.So(status.ID, convey.ShouldEqual, "1234")
		convey.So(status.Name, convey.ShouldEqual, "fake")
		convey.So(status.Message, convey.ShouldEqual, "init")
		convey.So(len(gc.gateway.Status.Nodes), convey.ShouldEqual, 1)
	})
}
