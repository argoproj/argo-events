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

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	fakeeventsclient "github.com/argoproj/argo-events/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

const (
	testNamespace = "test-ns"
)

var (
	fakeK8sClient    = fakeClient.NewSimpleClientset()
	fakeEventsClient = fakeeventsclient.NewSimpleClientset().ArgoprojV1alpha1()
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

func fakeEventBus() *aev1.EventBus {
	return &aev1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: aev1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      aev1.DefaultEventBusName,
		},
		Spec: aev1.EventBusSpec{
			NATS: &aev1.NATSBus{
				Native: &aev1.NativeStrategy{
					Auth: &aev1.AuthStrategyToken,
				},
			},
		},
	}
}

func fakeExoticEventBus() *aev1.EventBus {
	cID := "test-cluster-id"
	return &aev1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: aev1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-name",
		},
		Spec: aev1.EventBusSpec{
			NATS: &aev1.NATSBus{
				Exotic: &aev1.NATSConfig{
					ClusterID: &cID,
					URL:       "nats://adsaf:1234",
				},
			},
		},
	}
}

func fakeSensor() *aev1.Sensor {
	return &aev1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sensor",
			Namespace: testNamespace,
		},
		Spec: aev1.SensorSpec{
			Template: &aev1.Template{
				ServiceAccountName: "fake-sa",
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "test",
					},
				},
				Container: &aev1.Container{
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
			Triggers: []aev1.Trigger{
				{
					Template: &aev1.TriggerTemplate{
						Name: "fake-trigger",
						K8s: &aev1.StandardK8STrigger{
							Operation: "create",
							Source:    &aev1.ArtifactLocation{},
						},
					},
				},
			},
			Dependencies: []aev1.EventDependency{
				{
					Name:            "fake-dep",
					EventSourceName: "fake-source",
					EventName:       "fake-one",
				},
			},
		},
	}
}

func fakeCalendarEventSourceMap(name string) map[string]aev1.CalendarEventSource {
	return map[string]aev1.CalendarEventSource{name: {Schedule: "*/5 * * * *"}}
}

func fakeCalendarEventSource() *aev1.EventSource {
	return &aev1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-es",
		},
		Spec: aev1.EventSourceSpec{
			Calendar: fakeCalendarEventSourceMap("test"),
		},
	}
}

func TestGetValidator(t *testing.T) {
	t.Run("test get EventBus validator", func(t *testing.T) {
		byts, err := json.Marshal(fakeEventBus())
		assert.NoError(t, err)
		v, err := GetValidator(contextWithLogger(t), fakeK8sClient, fakeEventsClient, fromSchemaGVK(aev1.EventBusGroupVersionKind), nil, byts)
		assert.NoError(t, err)
		assert.NotNil(t, v)
	})
	t.Run("test get EventSource validator", func(t *testing.T) {
		byts, err := json.Marshal(fakeCalendarEventSource())
		assert.NoError(t, err)
		v, err := GetValidator(contextWithLogger(t), fakeK8sClient, fakeEventsClient, fromSchemaGVK(aev1.EventSourceGroupVersionKind), nil, byts)
		assert.NoError(t, err)
		assert.NotNil(t, v)
	})
	t.Run("test get Sensor validator", func(t *testing.T) {
		byts, err := json.Marshal(fakeSensor())
		assert.NoError(t, err)
		v, err := GetValidator(contextWithLogger(t), fakeK8sClient, fakeEventsClient, fromSchemaGVK(aev1.SensorGroupVersionKind), nil, byts)
		assert.NoError(t, err)
		assert.NotNil(t, v)
	})
}
