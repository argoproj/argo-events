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

package sensor

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestOperate(t *testing.T) {
	controller := getController()
	ctx := newSensorContext(sensorObj.DeepCopy(), controller)
	sensor, err := controller.sensorClient.ArgoprojV1alpha1().Sensors(sensorObj.Namespace).Create(sensorObj)
	assert.Nil(t, err)
	ctx.sensor = sensor.DeepCopy()

	tests := []struct {
		name       string
		updateFunc func()
		testFunc   func(oldMetadata *v1alpha1.SensorResources)
	}{
		{
			name:       "process a new sensor object",
			updateFunc: func() {},
			testFunc: func(oldMetadata *v1alpha1.SensorResources) {
				assert.NotNil(t, ctx.sensor.Status.Resources)
				metadata := ctx.sensor.Status.Resources
				deployment, err := controller.k8sClient.AppsV1().Deployments(metadata.Deployment.Namespace).Get(metadata.Deployment.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				service, err := controller.k8sClient.CoreV1().Services(metadata.Service.Namespace).Get(metadata.Service.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, v1alpha1.NodePhaseActive, ctx.sensor.Status.Phase)
				assert.Equal(t, 2, len(ctx.sensor.Status.Nodes))
				assert.Equal(t, "sensor is active", ctx.sensor.Status.Message)
			},
		},
		{
			name: "process a sensor object update",
			updateFunc: func() {
				ctx.sensor.Spec.Template.Spec.Containers[0].Name = "updated-name"
			},
			testFunc: func(oldMetadata *v1alpha1.SensorResources) {
				assert.NotNil(t, ctx.sensor.Status.Resources)
				metadata := ctx.sensor.Status.Resources
				deployment, err := controller.k8sClient.AppsV1().Deployments(metadata.Deployment.Namespace).Get(metadata.Deployment.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				assert.NotEqual(t, oldMetadata.Deployment.Annotations[common.AnnotationResourceSpecHash], deployment.Annotations[common.AnnotationResourceSpecHash])
				assert.Equal(t, deployment.Spec.Template.Spec.Containers[0].Name, "updated-name")
				service, err := controller.k8sClient.CoreV1().Services(metadata.Service.Namespace).Get(metadata.Service.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, oldMetadata.Service.Annotations[common.AnnotationResourceSpecHash], service.Annotations[common.AnnotationResourceSpecHash])
				assert.Equal(t, v1alpha1.NodePhaseActive, ctx.sensor.Status.Phase)
				assert.Equal(t, "sensor is updated", ctx.sensor.Status.Message)
			},
		},
		{
			name: "process a sensor in error state",
			updateFunc: func() {
				ctx.sensor.Status.Phase = v1alpha1.NodePhaseError
				ctx.sensor.Status.Message = "sensor is in error state"
				ctx.sensor.Spec.Template.Spec.Containers[0].Name = "revert-name"
			},
			testFunc: func(oldMetadata *v1alpha1.SensorResources) {
				assert.Equal(t, v1alpha1.NodePhaseActive, ctx.sensor.Status.Phase)
				assert.Equal(t, "sensor is active", ctx.sensor.Status.Message)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata := ctx.sensor.Status.Resources.DeepCopy()
			test.updateFunc()
			err := ctx.operate()
			assert.Nil(t, err)
			test.testFunc(metadata)
		})
	}
}

func TestUpdateSensorState(t *testing.T) {
	controller := getController()
	ctx := newSensorContext(sensorObj.DeepCopy(), controller)
	sensor, err := controller.sensorClient.ArgoprojV1alpha1().Sensors(sensorObj.Namespace).Create(sensorObj)
	assert.Nil(t, err)
	ctx.sensor = sensor.DeepCopy()
	assert.Equal(t, v1alpha1.NodePhaseNew, ctx.sensor.Status.Phase)
	ctx.sensor.Status.Phase = v1alpha1.NodePhaseActive
	ctx.updated = true
	ctx.updateSensorState()
	assert.Equal(t, v1alpha1.NodePhaseActive, ctx.sensor.Status.Phase)
}

func TestMarkSensorPhase(t *testing.T) {
	controller := getController()
	ctx := newSensorContext(sensorObj.DeepCopy(), controller)
	sensor, err := controller.sensorClient.ArgoprojV1alpha1().Sensors(sensorObj.Namespace).Create(sensorObj)
	assert.Nil(t, err)
	ctx.sensor = sensor.DeepCopy()
	ctx.markSensorPhase(v1alpha1.NodePhaseActive, false, "sensor is active")
	assert.Equal(t, v1alpha1.NodePhaseActive, ctx.sensor.Status.Phase)
	assert.Equal(t, "sensor is active", ctx.sensor.Status.Message)
}

func TestInitializeAllNodes(t *testing.T) {
	controller := getController()
	ctx := newSensorContext(sensorObj.DeepCopy(), controller)
	ctx.initializeAllNodes()
	for _, node := range ctx.sensor.Status.Nodes {
		assert.Equal(t, v1alpha1.NodePhaseNew, node.Phase)
		assert.NotEmpty(t, node.Name)
		assert.NotEmpty(t, node.ID)
	}
}

func TestMarkDependencyNodesActive(t *testing.T) {
	controller := getController()
	ctx := newSensorContext(sensorObj.DeepCopy(), controller)
	ctx.initializeAllNodes()
	ctx.markDependencyNodesActive()
	for _, node := range ctx.sensor.Status.Nodes {
		if node.Type == v1alpha1.NodeTypeEventDependency {
			assert.Equal(t, v1alpha1.NodePhaseActive, node.Phase)
		} else {
			assert.Equal(t, v1alpha1.NodePhaseNew, node.Phase)
		}
	}
}

func TestPersistUpdates(t *testing.T) {
	controller := getController()
	ctx := newSensorContext(sensorObj.DeepCopy(), controller)
	sensor, err := controller.sensorClient.ArgoprojV1alpha1().Sensors(sensorObj.Namespace).Create(sensorObj)
	assert.Nil(t, err)
	ctx.sensor = sensor.DeepCopy()
	ctx.sensor.Spec.Circuit = "fake-group"
	sensor, err = PersistUpdates(controller.sensorClient, ctx.sensor.DeepCopy(), ctx.logger)
	assert.Nil(t, err)
	assert.Equal(t, "fake-group", sensor.Spec.Circuit)
	assert.Equal(t, "fake-group", ctx.sensor.Spec.Circuit)
}
