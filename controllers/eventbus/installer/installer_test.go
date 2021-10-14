package installer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/common/logging"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestGetInstaller(t *testing.T) {
	t.Run("get installer", func(t *testing.T) {
		installer, err := getInstaller(testEventBus, nil, "", "", logging.NewArgoEventsLogger())
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok := installer.(*natsInstaller)
		assert.True(t, ok)

		installer, err = getInstaller(testExoticBus, nil, "", "", logging.NewArgoEventsLogger())
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok = installer.(*exoticNATSInstaller)
		assert.True(t, ok)
	})
}

func init() {
	_ = eventsourcev1alpha1.AddToScheme(scheme.Scheme)
	_ = sensorv1alpha1.AddToScheme(scheme.Scheme)
}

func TestGetLinkedEventSources(t *testing.T) {
	t.Run("get linked eventsources", func(t *testing.T) {
		es := fakeEmptyEventSource()
		es.Spec.EventBusName = "test-sa"
		es.Spec.Calendar = fakeCalendarEventSourceMap("test")
		cl := fake.NewClientBuilder().Build()
		ctx := context.Background()
		err := cl.Create(ctx, es, &client.CreateOptions{})
		assert.Nil(t, err)
		n, err := linkedEventSources(ctx, testNamespace, "test-sa", cl)
		assert.Nil(t, err)
		assert.Equal(t, n, 1)
	})
}

func TestGetLinkedSensors(t *testing.T) {
	t.Run("get linked sensors", func(t *testing.T) {
		s := fakeSensor()
		s.Spec.EventBusName = "test-sa"
		cl := fake.NewClientBuilder().Build()
		ctx := context.Background()
		err := cl.Create(ctx, s, &client.CreateOptions{})
		assert.Nil(t, err)
		n, err := linkedSensors(ctx, testNamespace, "test-sa", cl)
		assert.Nil(t, err)
		assert.Equal(t, n, 1)
	})
}

func fakeEmptyEventSource() *eventsourcev1alpha1.EventSource {
	return &eventsourcev1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-es",
		},
		Spec: eventsourcev1alpha1.EventSourceSpec{},
	}
}

func fakeCalendarEventSourceMap(name string) map[string]eventsourcev1alpha1.CalendarEventSource {
	return map[string]eventsourcev1alpha1.CalendarEventSource{name: {Schedule: "*/5 * * * *"}}
}

func fakeSensor() *sensorv1alpha1.Sensor {
	return &sensorv1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-sensor",
			Namespace: testNamespace,
		},
		Spec: sensorv1alpha1.SensorSpec{
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
}
