package eventsource

import (
	"context"
	"fmt"
	"strings"
	"testing"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	"github.com/stretchr/testify/assert"
)

const (
	testEventSourceName = "test-name"
	testNamespace       = "test-ns"
	testSecretName      = "test-secret"
	testSecretKey       = "test-secret-key"
	testConfigMapName   = "test-cm"
	testConfigMapKey    = "test-cm-key"
)

var (
	testSecretSelector = &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: testSecretName,
		},
		Key: testSecretKey,
	}

	testConfigMapSelector = &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: testConfigMapName,
		},
		Key: testConfigMapKey,
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
)

func fakeEmptyEventSource() *v1alpha1.EventSource {
	return &v1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testEventSourceName,
		},
		Spec: v1alpha1.EventSourceSpec{},
	}
}

func fakeCalendarEventSourceMap(name string) map[string]v1alpha1.CalendarEventSource {
	return map[string]v1alpha1.CalendarEventSource{name: {Schedule: "*/5 * * * *"}}
}

func fakeWebhookEventSourceMap(name string) map[string]v1alpha1.WebhookEventSource {
	return map[string]v1alpha1.WebhookEventSource{
		name: {
			WebhookContext: v1alpha1.WebhookContext{
				URL:      "http://a.b",
				Endpoint: "/abc",
				Port:     "1234",
			},
		},
	}
}

func fakeKafkaEventSourceMap(name string) map[string]v1alpha1.KafkaEventSource {
	return map[string]v1alpha1.KafkaEventSource{
		name: {
			URL:       "a.b",
			Partition: "abc",
			Topic:     "topic",
		},
	}
}

func fakeMQTTEventSourceMap(name string) map[string]v1alpha1.MQTTEventSource {
	return map[string]v1alpha1.MQTTEventSource{
		name: {
			URL:      "a.b",
			ClientID: "cid",
			Topic:    "topic",
		},
	}
}

func fakeHDFSEventSourceMap(name string) map[string]v1alpha1.HDFSEventSource {
	return map[string]v1alpha1.HDFSEventSource{
		name: {
			Type:                    "t",
			CheckInterval:           "aa",
			Addresses:               []string{"ad1"},
			HDFSUser:                "user",
			KrbCCacheSecret:         testSecretSelector,
			KrbKeytabSecret:         testSecretSelector,
			KrbUsername:             "user",
			KrbRealm:                "llss",
			KrbConfigConfigMap:      testConfigMapSelector,
			KrbServicePrincipalName: "name",
		},
	}
}

func init() {
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	_ = appv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
}

func TestReconcile(t *testing.T) {
	t.Run("test reconcile without eventbus", func(t *testing.T) {
		testEventSource := fakeEmptyEventSource()
		testEventSource.Spec.Calendar = fakeCalendarEventSourceMap("test")
		ctx := context.TODO()
		cl := fake.NewClientBuilder().Build()
		r := &reconciler{
			client:           cl,
			scheme:           scheme.Scheme,
			eventSourceImage: "test-image",
			logger:           logging.NewArgoEventsLogger(),
		}
		err := r.reconcile(ctx, testEventSource)
		assert.Error(t, err)
		assert.False(t, testEventSource.Status.IsReady())
	})

	t.Run("test reconcile with eventbus", func(t *testing.T) {
		testEventSource := fakeEmptyEventSource()
		testEventSource.Spec.Calendar = fakeCalendarEventSourceMap("test")
		ctx := context.TODO()
		cl := fake.NewClientBuilder().Build()
		testBus := fakeEventBusJetstream.DeepCopy()
		testBus.Status.MarkDeployed("test", "test")
		testBus.Status.MarkConfigured()
		err := cl.Create(ctx, testBus)
		assert.Nil(t, err)
		r := &reconciler{
			client:           cl,
			scheme:           scheme.Scheme,
			eventSourceImage: "test-image",
			logger:           logging.NewArgoEventsLogger(),
		}
		err = r.reconcile(ctx, testEventSource)
		assert.NoError(t, err)
		assert.True(t, testEventSource.Status.IsReady())
	})
}

func Test_BuildDeployment(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	r := &reconciler{
		client:           cl,
		scheme:           scheme.Scheme,
		eventSourceImage: "test-image",
		logger:           logging.NewArgoEventsLogger(),
	}
	testEventSource := fakeEmptyEventSource()
	testEventSource.Spec.HDFS = fakeHDFSEventSourceMap("test")
	testEventSource.Spec.Template = &v1alpha1.Template{
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "test",
			},
		},
		PriorityClassName: "test-class",
	}

	t.Run("test build HDFS", func(t *testing.T) {
		deployment, err := r.buildDeployment(fakeEventBus.DeepCopy(), testEventSource)
		assert.Nil(t, err)
		assert.NotNil(t, deployment)
		volumes := deployment.Spec.Template.Spec.Volumes
		assert.True(t, len(volumes) > 0)
		hasAuthVolume := false
		hasTmpVolume := false
		cmRefs, secretRefs := 0, 0
		for _, vol := range volumes {
			if vol.Name == "auth-volume" {
				hasAuthVolume = true
			}
			if vol.Name == "tmp" {
				hasTmpVolume = true
			}
			if strings.Contains(vol.Name, testEventSource.Spec.HDFS["test"].KrbCCacheSecret.Name) {
				secretRefs++
			}
			if strings.Contains(vol.Name, testEventSource.Spec.HDFS["test"].KrbConfigConfigMap.Name) {
				cmRefs++
			}
		}
		assert.True(t, hasAuthVolume)
		assert.True(t, hasTmpVolume)
		assert.True(t, len(deployment.Spec.Template.Spec.ImagePullSecrets) > 0)
		assert.True(t, cmRefs > 0)
		assert.True(t, secretRefs > 0)
		assert.Equal(t, deployment.Spec.Template.Spec.PriorityClassName, "test-class")
	})

	t.Run("test kafka eventbus secrets attached", func(t *testing.T) {
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

		deployment, err := r.buildDeployment(testBus, testEventSource)
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

		deployment, err := r.buildDeployment(fakeEventBusJetstreamWithTLS.DeepCopy(), testEventSource)
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
		webhooksWithSecrets := map[string]v1alpha1.WebhookEventSource{
			"webhook4": {
				WebhookContext: v1alpha1.WebhookContext{
					URL:      "http://a.b",
					Endpoint: "/webhook4",
					Port:     "1234",
					AuthSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "webhook4"},
						Key:                  "secret",
					},
				},
			},
			"webhook3": {
				WebhookContext: v1alpha1.WebhookContext{
					URL:      "http://a.b",
					Endpoint: "/webhook3",
					Port:     "1234",
					AuthSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "webhook3"},
						Key:                  "secret",
					},
				},
			},
			"webhook1": {
				WebhookContext: v1alpha1.WebhookContext{
					URL:      "http://a.b",
					Endpoint: "/webhook1",
					Port:     "1234",
					AuthSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "webhook1"},
						Key:                  "secret",
					},
				},
			},
			"webhook2": {
				WebhookContext: v1alpha1.WebhookContext{
					URL:      "http://a.b",
					Endpoint: "/webhook2",
					Port:     "1234",
					AuthSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "webhook2"},
						Key:                  "secret",
					},
				},
			},
		}
		es := testEventSource.DeepCopy()
		es.Spec.Webhook = webhooksWithSecrets

		wantVolumeNames := []string{"auth-volume", "cm-test-cm", "secret-test-secret", "secret-webhook1", "secret-webhook2", "secret-webhook3", "secret-webhook4", "tmp"}
		wantVolumeMountNames := []string{"auth-volume", "cm-test-cm", "secret-test-secret", "secret-webhook1", "secret-webhook2", "secret-webhook3", "secret-webhook4", "tmp"}

		deployment, err := r.buildDeployment(fakeEventBus, es)
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
