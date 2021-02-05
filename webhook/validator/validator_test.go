package validator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakeClient "k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	fakeeventbusclient "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/fake"
	fakeeventsourceclient "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/fake"
	fakesensorclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
)

const (
	testNamespace = "test-ns"
)

var (
	fakeK8sClient         = fakeClient.NewSimpleClientset()
	fakeEventBusClient    = fakeeventbusclient.NewSimpleClientset()
	fakeEventSourceClient = fakeeventsourceclient.NewSimpleClientset()
	fakeSensorClient      = fakesensorclient.NewSimpleClientset()
)

func contextWithLogger(t *testing.T) context.Context {
	t.Helper()
	return logging.WithLogger(context.Background(), logging.NewArgoEventsLogger())
}

func fromSchemaGVK(gvk schema.GroupVersionKind) metav1.GroupVersionKind {
	return metav1.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}
}

func fakeEventBus() *eventbusv1alpha1.EventBus {
	return &eventbusv1alpha1.EventBus{
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
	}
}

func fakeExoticEventBus() *eventbusv1alpha1.EventBus {
	cID := "test-cluster-id"
	return &eventbusv1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventbusv1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-name",
		},
		Spec: eventbusv1alpha1.EventBusSpec{
			NATS: &eventbusv1alpha1.NATSBus{
				Exotic: &eventbusv1alpha1.NATSConfig{
					ClusterID: &cID,
					URL:       "nats://adsaf:1234",
				},
			},
		},
	}
}

func fakeSensor() *sensorv1alpha1.Sensor {
	return &sensorv1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sensor",
			Namespace: testNamespace,
		},
		Spec: sensorv1alpha1.SensorSpec{
			Template: &sensorv1alpha1.Template{
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
			},
			Triggers: []sensorv1alpha1.Trigger{
				{
					Template: &sensorv1alpha1.TriggerTemplate{
						Name: "fake-trigger",
						K8s: &sensorv1alpha1.StandardK8STrigger{
							GroupVersionResource: metav1.GroupVersionResource{
								Group:    "k8s.io",
								Version:  "",
								Resource: "pods",
							},
							Operation: "create",
							Source:    &sensorv1alpha1.ArtifactLocation{},
						},
					},
				},
			},
			Dependencies: []sensorv1alpha1.EventDependency{
				{
					Name:            "fake-dep",
					EventSourceName: "fake-source",
					EventName:       "fake-one",
				},
			},
		},
	}
}

func fakeCalendarEventSourceMap(name string) map[string]eventsourcev1alpha1.CalendarEventSource {
	return map[string]eventsourcev1alpha1.CalendarEventSource{name: {Schedule: "*/5 * * * *"}}
}

func fakeCalendarEventSource() *eventsourcev1alpha1.EventSource {
	return &eventsourcev1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-es",
		},
		Spec: eventsourcev1alpha1.EventSourceSpec{
			Calendar: fakeCalendarEventSourceMap("test"),
		},
	}
}

func TestGetValidator(t *testing.T) {
	t.Run("test get EventBus validator", func(t *testing.T) {
		byts, err := json.Marshal(fakeEventBus())
		assert.NoError(t, err)
		v, err := GetValidator(contextWithLogger(t), fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, fromSchemaGVK(eventbusv1alpha1.SchemaGroupVersionKind), nil, byts)
		assert.NoError(t, err)
		assert.NotNil(t, v)
	})
	t.Run("test get EventSource validator", func(t *testing.T) {
		byts, err := json.Marshal(fakeCalendarEventSource())
		assert.NoError(t, err)
		v, err := GetValidator(contextWithLogger(t), fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, fromSchemaGVK(eventsourcev1alpha1.SchemaGroupVersionKind), nil, byts)
		assert.NoError(t, err)
		assert.NotNil(t, v)
	})
	t.Run("test get Sensor validator", func(t *testing.T) {
		byts, err := json.Marshal(fakeSensor())
		assert.NoError(t, err)
		v, err := GetValidator(contextWithLogger(t), fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, fromSchemaGVK(sensorv1alpha1.SchemaGroupVersionKind), nil, byts)
		assert.NoError(t, err)
		assert.NotNil(t, v)
	})
}
