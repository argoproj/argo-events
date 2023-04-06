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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	testNamespace      = "test-ns"
	authVolumeName     = "auth-volume"
	tmpVolumeName      = "tmp"
	testDataVolumeName = "test-data"
)

var (
	testLabels = map[string]string{"controller": "test-controller"}

	sensorObj = &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-sensor",
			Namespace: testNamespace,
		},
		Spec: v1alpha1.SensorSpec{
			Template: &v1alpha1.Template{
				ServiceAccountName: "fake-sa",
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "test",
					},
				},
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
				PriorityClassName: "test-class",
			},
			Triggers: []v1alpha1.Trigger{
				{
					Template: &v1alpha1.TriggerTemplate{
						Name: "fake-trigger",
						K8s: &v1alpha1.StandardK8STrigger{
							Operation: "create",
							Source:    &v1alpha1.ArtifactLocation{},
						},
					},
				},
			},
			Dependencies: []v1alpha1.EventDependency{
				{
					Name:            "fake-dep",
					EventSourceName: "fake-source",
					EventName:       "fake-one",
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
			Name:      common.DefaultEventBusName,
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

	fakeEventBusJetstream = &eventbusv1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventbusv1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      common.DefaultEventBusName,
		},
		Spec: eventbusv1alpha1.EventBusSpec{
			JetStream: &eventbusv1alpha1.JetStreamBus{
				Version: "x.x.x",
			},
		},
		Status: eventbusv1alpha1.EventBusStatus{
			Config: eventbusv1alpha1.BusConfig{
				JetStream: &eventbusv1alpha1.JetStreamConfig{
					URL: "nats://xxxx",
				},
			},
		},
	}

	fakeEventBusKafka = &eventbusv1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventbusv1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      common.DefaultEventBusName,
		},
		Spec: eventbusv1alpha1.EventBusSpec{
			Kafka: &eventbusv1alpha1.KafkaBus{
				URL: "localhost:9092",
			},
		},
		Status: eventbusv1alpha1.EventBusStatus{
			Config: eventbusv1alpha1.BusConfig{
				Kafka: &eventbusv1alpha1.KafkaBus{
					URL: "localhost:9092",
				},
			},
		},
	}
)

func Test_BuildDeployment(t *testing.T) {
	t.Run("test build with eventbus", func(t *testing.T) {
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}
		deployment, err := buildDeployment(args, fakeEventBus, nil)
		assert.Nil(t, err)
		assert.NotNil(t, deployment)
		volumes := deployment.Spec.Template.Spec.Volumes
		assert.True(t, len(volumes) > 0)
		hasAuthVolume := false
		hasTmpVolume := false
		hasTestDataVolume := false
		for _, vol := range volumes {
			if vol.Name == authVolumeName {
				hasAuthVolume = true
			}
			if vol.Name == tmpVolumeName {
				hasTmpVolume = true
			}
			if vol.Name == testDataVolumeName {
				hasTestDataVolume = true
			}
		}
		assert.True(t, hasAuthVolume)
		assert.True(t, hasTmpVolume)
		assert.True(t, hasTestDataVolume)
		assert.Len(t, volumes, 3, "Verify unexpected volumes aren't defined")
		assert.True(t, len(deployment.Spec.Template.Spec.ImagePullSecrets) > 0)
		assert.Equal(t, deployment.Spec.Template.Spec.PriorityClassName, "test-class")
		assert.Nil(t, deployment.Spec.RevisionHistoryLimit)

		hasSensorLiveReloadEnvVar := false
		hasSensorObjectEnvVar := false
		for _, env := range deployment.Spec.Template.Spec.Containers[0].Env {
			if env.Name == common.EnvVarSensorFilePath {
				hasSensorLiveReloadEnvVar = true
			}
			if env.Name == common.EnvVarSensorObject {
				hasSensorObjectEnvVar = true
			}
		}
		assert.True(t, hasSensorObjectEnvVar)
		assert.False(t, hasSensorLiveReloadEnvVar)
	})
	t.Run("test revisionHistoryLimit", func(t *testing.T) {
		sensorWithRevisionHistoryLimit := sensorObj.DeepCopy()
		sensorWithRevisionHistoryLimit.Spec.RevisionHistoryLimit = func() *int32 { i := int32(3); return &i }()
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorWithRevisionHistoryLimit,
			Labels: testLabels,
		}
		deployment, err := buildDeployment(args, fakeEventBus, nil)
		assert.Nil(t, err)
		assert.NotNil(t, deployment)
		assert.Equal(t, int32(3), *deployment.Spec.RevisionHistoryLimit)
	})
	t.Run("test liveReload", func(t *testing.T) {
		sensorWithLiveReload := sensorObj.DeepCopy()
		sensorWithLiveReload.Spec.LiveReload = true
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorWithLiveReload,
			Labels: testLabels,
		}
		deployment, err := buildDeployment(args, fakeEventBus, &corev1.ConfigMap{})
		assert.Nil(t, err)
		assert.NotNil(t, deployment)

		hasSensorLiveReloadEnvVar := false
		hasSensorObjectEnvVar := false
		for _, env := range deployment.Spec.Template.Spec.Containers[0].Env {
			if env.Name == common.EnvVarSensorFilePath {
				hasSensorLiveReloadEnvVar = true
			}
			if env.Name == common.EnvVarSensorObject {
				hasSensorObjectEnvVar = true
			}
		}
		assert.False(t, hasSensorObjectEnvVar)
		assert.True(t, hasSensorLiveReloadEnvVar)

		hasAuthVolume := false
		hasTmpVolume := false
		hasTestDataVolume := false
		hasConfigMapVolume := false
		for _, vol := range deployment.Spec.Template.Spec.Volumes {
			if vol.Name == authVolumeName {
				hasAuthVolume = true
			}
			if vol.Name == tmpVolumeName {
				hasTmpVolume = true
			}
			if vol.Name == testDataVolumeName {
				hasTestDataVolume = true
			}
			if vol.Name == "sensor-config-volume" {
				hasConfigMapVolume = true
			}
		}
		assert.True(t, hasAuthVolume)
		assert.True(t, hasTmpVolume)
		assert.True(t, hasTestDataVolume)
		assert.True(t, hasConfigMapVolume)
		assert.Len(t, deployment.Spec.Template.Spec.Volumes, 4, "Verify unexpected volumes aren't defined")

		hasAuthVolumeMount := false
		hasTmpVolumeMount := false
		hasTestDataVolumeMount := false
		hasConfigMapVolumeMount := false
		for _, volumeMount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
			if volumeMount.Name == authVolumeName {
				hasAuthVolumeMount = true
			}
			if volumeMount.Name == tmpVolumeName {
				hasTmpVolumeMount = true
			}
			if volumeMount.Name == testDataVolumeName {
				hasTestDataVolumeMount = true
			}
			if volumeMount.Name == "sensor-config-volume" {
				hasConfigMapVolumeMount = true
			}
		}
		assert.True(t, hasAuthVolumeMount)
		assert.True(t, hasTmpVolumeMount)
		assert.True(t, hasTestDataVolumeMount)
		assert.True(t, hasConfigMapVolumeMount)
		assert.Len(t, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, 4, "Verify unexpected volumes aren't mounted")
	})
	t.Run("test kafka eventbus secrets attached", func(t *testing.T) {
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}

		// add secrets to kafka eventbus
		testBus := fakeEventBusKafka.DeepCopy()
		testBus.Spec.Kafka.TLS = &apicommon.TLSConfig{
			CACertSecret: &corev1.SecretKeySelector{Key: "cert", LocalObjectReference: corev1.LocalObjectReference{Name: "tls-secret"}},
		}
		testBus.Spec.Kafka.SASL = &apicommon.SASLConfig{
			Mechanism:      "SCRAM-SHA-512",
			UserSecret:     &corev1.SecretKeySelector{Key: "username", LocalObjectReference: corev1.LocalObjectReference{Name: "sasl-secret"}},
			PasswordSecret: &corev1.SecretKeySelector{Key: "password", LocalObjectReference: corev1.LocalObjectReference{Name: "sasl-secret"}},
		}

		deployment, err := buildDeployment(args, testBus, nil)
		assert.Nil(t, err)
		assert.NotNil(t, deployment)

		hasSASLSecretVolume := false
		hasSASLSecretVolumeMount := false
		hasTLSSecretVolume := false
		hasTLSSecretVolumeMount := false
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Name == "secret-sasl-secret" {
				hasSASLSecretVolume = true
			}
			if volume.Name == "secret-tls-secret" {
				hasTLSSecretVolume = true
			}
		}
		for _, volumeMount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
			if volumeMount.Name == "secret-sasl-secret" {
				hasSASLSecretVolumeMount = true
			}
			if volumeMount.Name == "secret-tls-secret" {
				hasTLSSecretVolumeMount = true
			}
		}

		assert.True(t, hasSASLSecretVolume)
		assert.True(t, hasSASLSecretVolumeMount)
		assert.True(t, hasTLSSecretVolume)
		assert.True(t, hasTLSSecretVolumeMount)
	})
}

func TestResourceReconcile(t *testing.T) {
	t.Run("test resource reconcile without eventbus", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}
		err := Reconcile(cl, nil, args, logging.NewArgoEventsLogger())
		assert.Error(t, err)
		assert.False(t, sensorObj.Status.IsReady())
	})

	for _, eb := range []*eventbusv1alpha1.EventBus{fakeEventBus, fakeEventBusJetstream, fakeEventBusKafka} {
		testBus := eb.DeepCopy()

		t.Run("test resource reconcile with eventbus", func(t *testing.T) {
			ctx := context.TODO()
			cl := fake.NewClientBuilder().Build()
			testBus.Status.MarkDeployed("test", "test")
			testBus.Status.MarkConfigured()
			err := cl.Create(ctx, testBus)
			assert.Nil(t, err)
			args := &AdaptorArgs{
				Image:  testImage,
				Sensor: sensorObj,
				Labels: testLabels,
			}
			err = Reconcile(cl, testBus, args, logging.NewArgoEventsLogger())
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
	t.Run("test resource reconcile with live reload (create/update)", func(t *testing.T) {
		ctx := context.TODO()
		cl := fake.NewClientBuilder().Build()
		testBus := fakeEventBus.DeepCopy()
		testBus.Status.MarkDeployed("test", "test")
		testBus.Status.MarkConfigured()
		err := cl.Create(ctx, testBus)
		assert.Nil(t, err)
		testSensor := sensorObj.DeepCopy()
		testSensor.Spec.LiveReload = true
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: testSensor,
			Labels: testLabels,
		}
		err = Reconcile(cl, testBus, args, logging.NewArgoEventsLogger())
		assert.Nil(t, err)
		assert.True(t, testSensor.Status.IsReady())

		deployList := &appv1.DeploymentList{}
		err = cl.List(ctx, deployList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(deployList.Items))

		// Verify configmap created in sensor namespace and contents accurate
		cmList := &corev1.ConfigMapList{}
		err = cl.List(ctx, cmList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cmList.Items))
		liveReloadConfigMap := cmList.Items[0]
		assert.Equal(t, "sensor-fake-sensor", liveReloadConfigMap.Name)
		var sensorFromConfigMap v1alpha1.Sensor
		err = json.Unmarshal([]byte(liveReloadConfigMap.Data[common.SensorConfigMapFilename]), &sensorFromConfigMap)
		assert.Nil(t, err)
		assert.Equal(t, "fake-sensor", sensorFromConfigMap.Name)
		assert.Equal(t, "fake-dep", sensorFromConfigMap.Spec.Dependencies[0].Name)

		// Update the sensor dependencies and re-reconcile the sensor
		testSensor.Spec.Dependencies[0].Name = "updated-dep"
		err = Reconcile(cl, testBus, args, logging.NewArgoEventsLogger())
		assert.Nil(t, err)
		assert.True(t, testSensor.Status.IsReady())

		err = cl.List(ctx, deployList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(deployList.Items))

		// Verify configmap has been updated with new dependency
		err = cl.List(ctx, cmList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cmList.Items))
		liveReloadConfigMap = cmList.Items[0]
		assert.Equal(t, "sensor-fake-sensor", liveReloadConfigMap.Name)
		err = json.Unmarshal([]byte(liveReloadConfigMap.Data[common.SensorConfigMapFilename]), &sensorFromConfigMap)
		assert.Nil(t, err)
		assert.Equal(t, "fake-sensor", sensorFromConfigMap.Name)
		assert.Equal(t, "updated-dep", sensorFromConfigMap.Spec.Dependencies[0].Name)
	})
}
