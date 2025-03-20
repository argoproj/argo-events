/*
Copyright 2020 The Argoproj Authors.

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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
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
			Template: &v1alpha1.Template{
				ServiceAccountName: "fake-sa",
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "test",
					},
				},
				Container: &v1alpha1.Container{
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

	fakeEventBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      v1alpha1.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			NATS: &v1alpha1.NATSBus{
				Native: &v1alpha1.NativeStrategy{
					Auth: &v1alpha1.AuthStrategyToken,
				},
			},
		},
		Status: v1alpha1.EventBusStatus{
			Config: v1alpha1.BusConfig{
				NATS: &v1alpha1.NATSConfig{
					URL:  "nats://xxxx",
					Auth: &v1alpha1.AuthStrategyToken,
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

	fakeEventBusJetstream = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      v1alpha1.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStream: &v1alpha1.JetStreamBus{
				Version: "x.x.x",
			},
		},
		Status: v1alpha1.EventBusStatus{
			Config: v1alpha1.BusConfig{
				JetStream: &v1alpha1.JetStreamConfig{
					URL: "nats://xxxx",
				},
			},
		},
	}

	fakeEventBusJetstreamWithTLS = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      v1alpha1.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStream: &v1alpha1.JetStreamBus{},
		},
		Status: v1alpha1.EventBusStatus{
			Config: v1alpha1.BusConfig{
				JetStream: &v1alpha1.JetStreamConfig{
					URL: "nats://xxxx",
					TLS: &v1alpha1.TLSConfig{
						CACertSecret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secret-0",
							},
							Key: "ca.crt",
						},
						ClientKeySecret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secret-1",
							},
							Key: "tls.key",
						},
						ClientCertSecret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secret-1",
							},
							Key: "tls.crt",
						},
					},
				},
			},
		},
	}

	fakeEventBusKafka = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      v1alpha1.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			Kafka: &v1alpha1.KafkaBus{
				URL: "localhost:9092",
			},
		},
		Status: v1alpha1.EventBusStatus{
			Config: v1alpha1.BusConfig{
				Kafka: &v1alpha1.KafkaBus{
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
		deployment, err := buildDeployment(args, fakeEventBus)
		assert.Nil(t, err)
		assert.NotNil(t, deployment)
		volumes := deployment.Spec.Template.Spec.Volumes
		assert.True(t, len(volumes) > 0)
		hasAuthVolume := false
		hasTmpVolume := false
		for _, vol := range volumes {
			if vol.Name == "auth-volume" {
				hasAuthVolume = true
			}
			if vol.Name == "tmp" {
				hasTmpVolume = true
			}
		}
		assert.True(t, hasAuthVolume)
		assert.True(t, hasTmpVolume)
		assert.True(t, len(deployment.Spec.Template.Spec.ImagePullSecrets) > 0)
		assert.Equal(t, deployment.Spec.Template.Spec.PriorityClassName, "test-class")
		assert.Nil(t, deployment.Spec.RevisionHistoryLimit)
	})
	t.Run("test revisionHistoryLimit", func(t *testing.T) {
		sensorWithRevisionHistoryLimit := sensorObj.DeepCopy()
		sensorWithRevisionHistoryLimit.Spec.RevisionHistoryLimit = func() *int32 { i := int32(3); return &i }()
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorWithRevisionHistoryLimit,
			Labels: testLabels,
		}
		deployment, err := buildDeployment(args, fakeEventBus)
		assert.Nil(t, err)
		assert.NotNil(t, deployment)
		assert.Equal(t, int32(3), *deployment.Spec.RevisionHistoryLimit)
	})

	t.Run("test kafka eventbus secrets attached", func(t *testing.T) {
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}

		// add secrets to kafka eventbus
		testBus := fakeEventBusKafka.DeepCopy()
		testBus.Spec.Kafka.TLS = &v1alpha1.TLSConfig{
			CACertSecret: &corev1.SecretKeySelector{Key: "cert", LocalObjectReference: corev1.LocalObjectReference{Name: "tls-secret"}},
		}
		testBus.Spec.Kafka.SASL = &v1alpha1.SASLConfig{
			Mechanism:      "SCRAM-SHA-512",
			UserSecret:     &corev1.SecretKeySelector{Key: "username", LocalObjectReference: corev1.LocalObjectReference{Name: "sasl-secret"}},
			PasswordSecret: &corev1.SecretKeySelector{Key: "password", LocalObjectReference: corev1.LocalObjectReference{Name: "sasl-secret"}},
		}

		deployment, err := buildDeployment(args, testBus)
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

	t.Run("test jetstream eventbus secrets attached", func(t *testing.T) {
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}

		deployment, err := buildDeployment(args, fakeEventBusJetstreamWithTLS.DeepCopy())
		assert.Nil(t, err)
		assert.NotNil(t, deployment)

		hasCAVolume := false
		hasCertVolume := false
		hasCAVolumeMount := false
		hasCertVolumeMount := false
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Name == "secret-0" {
				if hasCAVolume {
					assert.Fail(t, "Secrets should be de-duplicated")
				}
				hasCAVolume = true
			}
			if volume.Name == "secret-1" {
				if hasCertVolume {
					assert.Fail(t, "Secrets should be de-duplicated")
				}
				hasCertVolume = true
			}
		}
		for _, volumeMount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
			if volumeMount.Name == "secret-0" {
				hasCAVolumeMount = true
				assert.Equal(t, volumeMount.MountPath, fmt.Sprintf("/argo-events/secrets/%s", volumeMount.Name))
			}
			if volumeMount.Name == "secret-1" {
				hasCertVolumeMount = true
				assert.Equal(t, volumeMount.MountPath, fmt.Sprintf("/argo-events/secrets/%s", volumeMount.Name))
			}
		}

		assert.True(t, hasCAVolume)
		assert.True(t, hasCertVolume)
		assert.True(t, hasCAVolumeMount)
		assert.True(t, hasCertVolumeMount)
	})

	t.Run("test secret volume and volumemount order deterministic", func(t *testing.T) {
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}

		wantVolumeNames := []string{"test-data", "auth-volume", "tmp"}
		wantVolumeMountNames := []string{"test-data", "auth-volume", "tmp"}

		deployment, err := buildDeployment(args, fakeEventBus)
		assert.Nil(t, err)
		assert.NotNil(t, deployment)
		gotVolumes := deployment.Spec.Template.Spec.Volumes
		gotVolumeMounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts

		var gotVolumeNames []string
		var gotVolumeMountNames []string

		for _, v := range gotVolumes {
			gotVolumeNames = append(gotVolumeNames, v.Name)
		}
		for _, v := range gotVolumeMounts {
			gotVolumeMountNames = append(gotVolumeMountNames, v.Name)
		}

		assert.Equal(t, len(gotVolumes), len(wantVolumeNames))
		assert.Equal(t, len(gotVolumeMounts), len(wantVolumeMountNames))

		for i := range gotVolumeNames {
			assert.Equal(t, gotVolumeNames[i], wantVolumeNames[i])
		}
		for i := range gotVolumeMountNames {
			assert.Equal(t, gotVolumeMountNames[i], wantVolumeMountNames[i])
		}
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

	for _, eb := range []*v1alpha1.EventBus{fakeEventBus, fakeEventBusJetstream, fakeEventBusKafka} {
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
}
