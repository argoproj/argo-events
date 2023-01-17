package installer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/controllers"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	testJetStreamImage         = "test-js-image"
	testJSReloaderImage        = "test-nats-rl-image"
	testJetStreamExporterImage = "test-js-e-image"
)

var (
	fakeConfig = &controllers.GlobalConfig{
		EventBus: &controllers.EventBusConfig{
			NATS: &controllers.StanConfig{
				Versions: []controllers.StanVersion{
					{
						Version:              "0.22.1",
						NATSStreamingImage:   "test-n-s-image",
						MetricsExporterImage: "test-n-s-m-image",
					},
				},
			},
			JetStream: &controllers.JetStreamConfig{
				Versions: []controllers.JetStreamVersion{
					{
						Version:              "2.7.3",
						NatsImage:            testJetStreamImage,
						ConfigReloaderImage:  testJSReloaderImage,
						MetricsExporterImage: testJetStreamExporterImage,
					},
				},
			},
		},
	}
)

func TestGetInstaller(t *testing.T) {
	t.Run("get installer", func(t *testing.T) {
		installer, err := getInstaller(testNatsEventBus, nil, nil, fakeConfig, zaptest.NewLogger(t).Sugar())
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok := installer.(*natsInstaller)
		assert.True(t, ok)

		installer, err = getInstaller(testNatsExoticBus, nil, nil, fakeConfig, zaptest.NewLogger(t).Sugar())
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok = installer.(*exoticNATSInstaller)
		assert.True(t, ok)
	})

	t.Run("get jetstream installer", func(t *testing.T) {
		installer, err := getInstaller(testJetStreamEventBus, nil, nil, fakeConfig, zaptest.NewLogger(t).Sugar())
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok := installer.(*jetStreamInstaller)
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
			Triggers: []sensorv1alpha1.Trigger{
				{
					Template: &sensorv1alpha1.TriggerTemplate{
						Name: "fake-trigger",
						K8s: &sensorv1alpha1.StandardK8STrigger{
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

func TestInstall(t *testing.T) {
	kubeClient := k8sfake.NewSimpleClientset()
	cl := fake.NewClientBuilder().Build()
	ctx := context.TODO()

	t.Run("test nats error", func(t *testing.T) {
		testObj := testNatsEventBus.DeepCopy()
		testObj.Spec.NATS = nil
		err := Install(ctx, testObj, cl, kubeClient, fakeConfig, zaptest.NewLogger(t).Sugar())
		assert.Error(t, err)
		assert.Equal(t, "invalid eventbus spec", err.Error())
	})

	t.Run("test nats install ok", func(t *testing.T) {
		testObj := testNatsEventBus.DeepCopy()
		err := Install(ctx, testObj, cl, kubeClient, fakeConfig, zaptest.NewLogger(t).Sugar())
		assert.NoError(t, err)
		assert.True(t, testObj.Status.IsReady())
		assert.NotNil(t, testObj.Status.Config.NATS)
		assert.NotEmpty(t, testObj.Status.Config.NATS.URL)
		assert.NotNil(t, testObj.Status.Config.NATS.Auth)
		assert.NotNil(t, testObj.Status.Config.NATS.AccessSecret)
	})

	t.Run("test jetstream error", func(t *testing.T) {
		testObj := testJetStreamEventBus.DeepCopy()
		testObj.Spec.JetStream = nil
		err := Install(ctx, testObj, cl, kubeClient, fakeConfig, zaptest.NewLogger(t).Sugar())
		assert.Error(t, err)
		assert.Equal(t, "invalid eventbus spec", err.Error())
	})

	t.Run("test jetstream install ok", func(t *testing.T) {
		testObj := testJetStreamEventBus.DeepCopy()
		err := Install(ctx, testObj, cl, kubeClient, fakeConfig, zaptest.NewLogger(t).Sugar())
		assert.NoError(t, err)
		assert.True(t, testObj.Status.IsReady())
		assert.NotNil(t, testObj.Status.Config.JetStream)
		assert.NotEmpty(t, testObj.Status.Config.JetStream.URL)
		assert.NotNil(t, testObj.Status.Config.JetStream.AccessSecret)
	})
}
