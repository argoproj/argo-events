package eventsource

import (
	"context"
	"testing"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
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
	_ = eventbusv1alpha1.AddToScheme(scheme.Scheme)
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
		testBus := fakeEventBus.DeepCopy()
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
