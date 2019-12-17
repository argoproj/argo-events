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
	"testing"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var sensorObj = &v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor",
		Namespace: "faker",
	},
	Spec: v1alpha1.SensorSpec{
		Template: &corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-sensor",
				Namespace: "faker",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "fake-sensor",
						ImagePullPolicy: corev1.PullAlways,
						Image:           "argoproj/sensor",
					},
				},
			},
		},
		EventProtocol: &apicommon.EventProtocol{
			Http: apicommon.Http{
				Port: "12000",
			},
			Type: apicommon.HTTP,
		},
		Triggers: []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					GroupVersionResource: &metav1.GroupVersionResource{
						Group:    "k8s.io",
						Version:  "",
						Resource: "pods",
					},
					Source: &v1alpha1.ArtifactLocation{},
				},
			},
		},
		Dependencies: []v1alpha1.EventDependency{
			{
				Name: "fake-gateway:fake-one",
			},
		},
	},
}

func TestResource_BuildService(t *testing.T) {
	controller := getController()
	opctx := newSensorContext(sensorObj.DeepCopy(), controller)
	service, err := opctx.serviceBuilder()
	assert.Nil(t, err)
	assert.NotNil(t, service)
	assert.NotEmpty(t, service.Annotations[common.AnnotationResourceSpecHash])
}

func TestResource_BuildServiceWithLabelsAnnotations(t *testing.T) {
	controller := getController()
	sensorCopy := sensorObj.DeepCopy()

	sensorCopy.Spec.EventProtocol.Http.Labels = map[string]string{}
	sensorCopy.Spec.EventProtocol.Http.Labels["test-label"] = "label1"
	sensorCopy.Spec.EventProtocol.Http.Annotations = map[string]string{}
	sensorCopy.Spec.EventProtocol.Http.Annotations["test-annotation"] = "annotation1"

	opctx := newSensorContext(sensorCopy, controller)
	service, err := opctx.serviceBuilder()
	assert.Nil(t, err)
	assert.NotNil(t, service)
	assert.NotEmpty(t, service.Annotations[common.AnnotationResourceSpecHash])
	assert.NotNil(t, service.ObjectMeta.Labels)
	assert.NotNil(t, service.ObjectMeta.Annotations)
	assert.Equal(t, service.ObjectMeta.Labels["test-label"], "label1")
	assert.Equal(t, service.ObjectMeta.Annotations["test-annotation"], "annotation1")
}

func TestResource_BuildDeployment(t *testing.T) {
	controller := getController()
	opctx := newSensorContext(sensorObj.DeepCopy(), controller)
	deployment, err := opctx.deploymentBuilder()
	assert.Nil(t, err)
	assert.NotNil(t, deployment)
	assert.NotEmpty(t, deployment.Annotations[common.AnnotationResourceSpecHash])
	assert.Equal(t, int(*deployment.Spec.Replicas), 1)
}

func TestResource_SetupContainers(t *testing.T) {
	controller := getController()
	opctx := newSensorContext(sensorObj.DeepCopy(), controller)
	deployment, err := opctx.deploymentBuilder()
	assert.Nil(t, err)
	assert.NotNil(t, deployment)
	assert.Equal(t, deployment.Spec.Template.Spec.Containers[0].Env[0].Name, common.SensorName)
	assert.Equal(t, deployment.Spec.Template.Spec.Containers[0].Env[0].Value, opctx.sensor.Name)
	assert.Equal(t, deployment.Spec.Template.Spec.Containers[0].Env[1].Name, common.SensorNamespace)
	assert.Equal(t, deployment.Spec.Template.Spec.Containers[0].Env[1].Value, opctx.sensor.Namespace)
	assert.Equal(t, deployment.Spec.Template.Spec.Containers[0].Env[2].Name, common.EnvVarControllerInstanceID)
	assert.Equal(t, deployment.Spec.Template.Spec.Containers[0].Env[2].Value, controller.Config.InstanceID)
}

func TestResource_UpdateResources(t *testing.T) {
	controller := getController()
	ctx := newSensorContext(sensorObj.DeepCopy(), controller)
	err := ctx.createSensorResources()
	assert.Nil(t, err)

	tests := []struct {
		name       string
		updateFunc func()
		testFunc   func(t *testing.T, oldResources *v1alpha1.SensorResources)
	}{
		{
			name: "update deployment when sensor template is updated",
			updateFunc: func() {
				ctx.sensor.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
			},
			testFunc: func(t *testing.T, oldResources *v1alpha1.SensorResources) {
				oldDeployment := oldResources.Deployment
				deployment, err := ctx.controller.k8sClient.AppsV1().Deployments(ctx.sensor.Status.Resources.Deployment.Namespace).Get(ctx.sensor.Status.Resources.Deployment.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				assert.NotEqual(t, oldDeployment.Annotations[common.AnnotationResourceSpecHash], deployment.Annotations[common.AnnotationResourceSpecHash])

				oldService := oldResources.Service
				service, err := ctx.controller.k8sClient.CoreV1().Services(ctx.sensor.Status.Resources.Service.Namespace).Get(ctx.sensor.Status.Resources.Service.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, oldService.Annotations[common.AnnotationResourceSpecHash], service.Annotations[common.AnnotationResourceSpecHash])
			},
		},
		{
			name: "update event protocol to NATS and check the service deletion",
			updateFunc: func() {
				ctx.sensor.Spec.EventProtocol.Type = apicommon.NATS
			},
			testFunc: func(t *testing.T, oldResources *v1alpha1.SensorResources) {
				oldDeployment := oldResources.Deployment
				deployment, err := ctx.controller.k8sClient.AppsV1().Deployments(ctx.sensor.Status.Resources.Deployment.Namespace).Get(ctx.sensor.Status.Resources.Deployment.Name, metav1.GetOptions{})
				assert.Nil(t, err)
				assert.NotNil(t, deployment)
				assert.Equal(t, oldDeployment.Annotations[common.AnnotationResourceSpecHash], deployment.Annotations[common.AnnotationResourceSpecHash])

				oldService := oldResources.Service
				service, err := ctx.controller.k8sClient.CoreV1().Services(oldService.Namespace).Get(oldService.Name, metav1.GetOptions{})
				assert.NotNil(t, err)
				assert.Nil(t, service)
			},
		},
	}

	for _, test := range tests {
		oldResources := ctx.sensor.Status.Resources.DeepCopy()
		t.Run(test.name, func(t *testing.T) {
			test.updateFunc()
			err := ctx.updateSensorResources()
			assert.Nil(t, err)
			test.testFunc(t, oldResources)
		})
	}
}
