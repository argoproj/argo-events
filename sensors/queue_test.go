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
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorFake "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	"github.com/argoproj/argo-events/sensors/types"
)

func TestProcessQueue(t *testing.T) {
	sensorClient := sensorFake.NewSimpleClientset()
	dynamicClient := dfake.NewSimpleDynamicClient(runtime.NewScheme())
	k8sClient := fake.NewSimpleClientset()
	obj := sensorObj.DeepCopy()

	newObj, err := sensorClient.ArgoprojV1alpha1().Sensors(obj.Namespace).Create(obj)
	assert.Nil(t, err)

	obj = newObj.DeepCopy()
	sensorCtx := NewSensorContext(sensorClient, k8sClient, dynamicClient, obj, "1")

	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			ID:              "1",
			Source:          "webhook-gateway",
			SpecVersion:     cloudevents.VersionV1,
			Type:            "webhook",
			DataContentType: common.MediaTypeJSON,
			Subject:         "example-1",
			Time:            v1.Time{Time: time.Now().UTC()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"} }"),
	}

	group1 := obj.NodeID("group1")
	dep1 := obj.NodeID("dep1")
	obj.Spec.DependencyGroups = []v1alpha1.DependencyGroup{
		{
			Name: "group1",
			Dependencies: []string{
				"dep1",
			},
		},
	}
	obj.Spec.Dependencies = []v1alpha1.EventDependency{
		{
			Name:        "dep1",
			GatewayName: "webhook-gateway",
			EventName:   "example-1",
		},
	}
	obj.Status.Nodes = map[string]v1alpha1.NodeStatus{
		group1: {
			ID:          group1,
			Name:        "group1",
			DisplayName: "group1",
			Type:        v1alpha1.NodeTypeDependencyGroup,
		},
		dep1: {
			ID:          dep1,
			Name:        "dep1",
			DisplayName: "dep1",
			Type:        v1alpha1.NodeTypeEventDependency,
			Phase:       v1alpha1.NodePhaseActive,
		},
	}

	deployment := newUnstructured("apps/v1", "Deployment", "fake", "fake-deployment")
	artifact, err := v1alpha1.NewResourceArtifact(deployment)
	assert.NoError(t, err)
	obj.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: artifact,
	}

	sensorCtx.processQueue(&types.Notification{
		Event:            event,
		EventDependency:  &obj.Spec.Dependencies[0],
		Sensor:           obj,
		NotificationType: v1alpha1.EventNotification,
	})

	assert.Equal(t, sensorCtx.Sensor.Status.TriggerCycleStatus, v1alpha1.TriggerCycleSuccess)
	assert.Equal(t, int32(1), sensorCtx.Sensor.Status.TriggerCycleCount)

	sensorCtx.Sensor.Status.TriggerCycleStatus = v1alpha1.TriggerCycleFailure
	sensorCtx.Sensor.Spec.ErrorOnFailedRound = true
	sensorCtx.processQueue(&types.Notification{
		Event:            event,
		EventDependency:  &obj.Spec.Dependencies[0],
		NotificationType: v1alpha1.EventNotification,
	})
	assert.Equal(t, int32(1), sensorCtx.Sensor.Status.TriggerCycleCount)
}
