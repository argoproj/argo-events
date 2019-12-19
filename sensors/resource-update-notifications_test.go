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

package sensors

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperateResourceUpdateNotification(t *testing.T) {
	obj := sensorObj.DeepCopy()
	dep1 := obj.NodeID("dep1")
	dep2 := obj.NodeID("dep2")

	obj.Spec.Dependencies = []v1alpha1.EventDependency{
		{
			Name:        "dep1",
			GatewayName: "webhook-gateway",
			EventName:   "example-1",
		},
		{
			Name:        "dep2",
			GatewayName: "webhook-gateway",
			EventName:   "example-2",
		},
	}
	obj.Status = v1alpha1.SensorStatus{
		Nodes: map[string]v1alpha1.NodeStatus{
			dep1: {
				ID:   dep1,
				Name: "dep1",
				Type: v1alpha1.NodeTypeEventDependency,
			},
			dep2: {
				ID:   dep2,
				Name: "dep2",
				Type: v1alpha1.NodeTypeEventDependency,
			},
		},
	}

	sensorCtx := &SensorContext{
		Sensor: obj.DeepCopy(),
		Logger: common.NewArgoEventsLogger(),
	}

	tests := []struct {
		name       string
		updateFunc func()
		testFunc   func()
	}{
		{
			name: "sensor update with dependencies removed",
			updateFunc: func() {
				obj.Spec.Dependencies = obj.Spec.Dependencies[:len(obj.Spec.Dependencies)-1]
			},
			testFunc: func() {
				assert.Empty(t, sensorCtx.Sensor.Status.Nodes[dep2])
			},
		},
		{
			name: "a dependency is added",
			updateFunc: func() {
				obj.Spec.Dependencies = append()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.updateFunc()
			sensorCtx.operateResourceUpdateNotification(&types.Notification{
				Event:            nil,
				EventDependency:  nil,
				Sensor:           obj,
				NotificationType: v1alpha1.ResourceUpdateNotification,
			})
			test.testFunc()
		})
	}
}
