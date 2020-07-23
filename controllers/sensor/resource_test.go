/*
Copyright 2020 BlackRock, Inc.

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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/common/logging"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	testNamespace = "test-ns"
)

var (
	testLabels = map[string]string{"controller": "test-controller"}

	sensorObj = &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-sensor",
			Namespace: testNamespace,
		},
		Spec: v1alpha1.SensorSpec{
			Template: v1alpha1.Template{
				ServiceAccountName: "fake-sa",
				Container: &corev1.Container{
					VolumeMounts: []corev1.VolumeMount{
						{
							MountPath: "/test-data",
							Name:      "test-data",
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "test-data",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
			Triggers: []v1alpha1.Trigger{
				{
					Template: &v1alpha1.TriggerTemplate{
						Name: "fake-trigger",
						K8s: &v1alpha1.StandardK8STrigger{
							GroupVersionResource: metav1.GroupVersionResource{
								Group:    "k8s.io",
								Version:  "",
								Resource: "pods",
							},
							Operation: "create",
							Source:    &v1alpha1.ArtifactLocation{},
						},
					},
				},
			},
			Dependencies: []v1alpha1.EventDependency{
				{
					Name:        "fake-dep",
					GatewayName: "fake-gateway",
					EventName:   "fake-one",
				},
			},
		},
	}

	sensorObjNoTemplate = &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-sensor",
			Namespace: testNamespace,
		},
		Spec: v1alpha1.SensorSpec{
			Triggers: []v1alpha1.Trigger{
				{
					Template: &v1alpha1.TriggerTemplate{
						Name: "fake-trigger",
						K8s: &v1alpha1.StandardK8STrigger{
							GroupVersionResource: metav1.GroupVersionResource{
								Group:    "k8s.io",
								Version:  "",
								Resource: "pods",
							},
							Operation: "create",
							Source:    &v1alpha1.ArtifactLocation{},
						},
					},
				},
			},
			Dependencies: []v1alpha1.EventDependency{
				{
					Name:        "fake-dep",
					GatewayName: "fake-gateway",
					EventName:   "fake-one",
				},
			},
		},
	}

	fakeEventBus = &eventbusv1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventbusv1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "default",
		},
		Spec: eventbusv1alpha1.EventBusSpec{
			NATS: &eventbusv1alpha1.NATSBus{
				Native: &eventbusv1alpha1.NativeStrategy{
					Auth: &eventbusv1alpha1.AuthStrategyToken,
				},
			},
		},
		Status: eventbusv1alpha1.EventBusStatus{
			Config: eventbusv1alpha1.BusConfig{
				NATS: &eventbusv1alpha1.NATSConfig{
					URL:  "nats://xxxx",
					Auth: &eventbusv1alpha1.AuthStrategyToken,
					AccessSecret: &corev1.SecretKeySelector{
						Key: "test-key",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-name",
						},
					},
				},
			},
		},
	}
)

func Test_BuildDeployment(t *testing.T) {
	sensorObjs := []*v1alpha1.Sensor{sensorObj, sensorObjNoTemplate}
	t.Run("test build with eventbus", func(t *testing.T) {
		for _, sObj := range sensorObjs {
			args := &AdaptorArgs{
				Image:  testImage,
				Sensor: sObj,
				Labels: testLabels,
			}
			deployment, err := buildDeployment(args, fakeEventBus)
			assert.Nil(t, err)
			assert.NotNil(t, deployment)
			volumes := deployment.Spec.Template.Spec.Volumes
			assert.True(t, len(volumes) > 0)
			hasAuthVolume := false
			for _, vol := range volumes {
				if vol.Name == "auth-volume" {
					hasAuthVolume = true
					break
				}
			}
			assert.True(t, hasAuthVolume)
		}
	})
}

func TestResourceReconcile(t *testing.T) {
	t.Run("test resource reconcile without eventbus", func(t *testing.T) {
		cl := fake.NewFakeClient(sensorObj)
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}
		err := Reconcile(cl, args, logging.NewArgoEventsLogger())
		assert.Error(t, err)
		assert.False(t, sensorObj.Status.IsReady())
	})

	t.Run("test resource reconcile with eventbus", func(t *testing.T) {
		ctx := context.TODO()
		cl := fake.NewFakeClient(sensorObj)
		testBus := fakeEventBus.DeepCopy()
		testBus.Status.MarkDeployed("test", "test")
		testBus.Status.MarkConfigured()
		err := cl.Create(ctx, testBus)
		assert.Nil(t, err)
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}
		err = Reconcile(cl, args, logging.NewArgoEventsLogger())
		assert.Nil(t, err)
		assert.True(t, sensorObj.Status.IsReady())

		deployList := &appv1.DeploymentList{}
		err = cl.List(ctx, deployList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(deployList.Items))

		svcList := &corev1.ServiceList{}
		err = cl.List(ctx, svcList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(svcList.Items))
	})
}
