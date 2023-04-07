package fixtures

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	eventbuspkg "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/typed/eventbus/v1alpha1"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/typed/eventsource/v1alpha1"
	sensorpkg "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
	testutil "github.com/argoproj/argo-events/test/util"
)

type When struct {
	t                 *testing.T
	eventBusClient    eventbuspkg.EventBusInterface
	eventSourceClient eventsourcepkg.EventSourceInterface
	sensorClient      sensorpkg.SensorInterface
	eventBus          *eventbusv1alpha1.EventBus
	eventSource       *eventsourcev1alpha1.EventSource
	sensor            *sensorv1alpha1.Sensor
	restConfig        *rest.Config
	kubeClient        kubernetes.Interface
}

func (w *When) CreateEventBus() *When {
	w.t.Helper()
	if w.eventBus == nil {
		w.t.Fatal("No event bus to create")
	}
	w.t.Log("Creating event bus", w.eventBus.Name)
	ctx := context.Background()
	eb, err := w.eventBusClient.Create(ctx, w.eventBus, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.eventBus = eb
	}
	return w
}

func (w *When) DeleteEventBus() *When {
	w.t.Helper()
	if w.eventBus == nil {
		w.t.Fatal("No event bus to delete")
	}
	w.t.Log("Deleting event bus", w.eventBus.Name)
	ctx := context.Background()
	err := w.eventBusClient.Delete(ctx, w.eventBus.Name, metav1.DeleteOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) CreateEventSource() *When {
	w.t.Helper()
	if w.eventSource == nil {
		w.t.Fatal("No event source to create")
	}
	w.t.Log("Creating event source", w.eventSource.Name)
	ctx := context.Background()
	es, err := w.eventSourceClient.Create(ctx, w.eventSource, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.eventSource = es
	}
	return w
}

func (w *When) DeleteEventSource() *When {
	w.t.Helper()
	if w.eventSource == nil {
		w.t.Fatal("No event source to delete")
	}
	w.t.Log("Deleting event source", w.eventSource.Name)
	ctx := context.Background()
	err := w.eventSourceClient.Delete(ctx, w.eventSource.Name, metav1.DeleteOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) CreateSensor() *When {
	w.t.Helper()
	if w.sensor == nil {
		w.t.Fatal("No sensor to create")
	}
	w.t.Log("Creating sensor", w.sensor.Name)
	ctx := context.Background()
	s, err := w.sensorClient.Create(ctx, w.sensor, metav1.CreateOptions{})
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.sensor = s
	}
	return w
}

func (w *When) UpdateSensor(g *Given) *When {
	w.t.Helper()
	if w.sensor == nil {
		w.t.Fatal("No sensor to update")
	}
	w.t.Log("Update sensor", w.sensor.Name)
	ctx := context.Background()

	sensor, err := w.sensorClient.Get(ctx, w.sensor.Name, metav1.GetOptions{})
	if err != nil {
		w.t.Fatal(err)
	}

	// update spec only
	sensor.Spec = g.sensor.Spec

	s, err := w.sensorClient.Update(ctx, sensor, metav1.UpdateOptions{})
	if err != nil {
		w.t.Fatal(err)
	} else {
		w.sensor = s
	}
	return w
}

func (w *When) DeleteSensor() *When {
	w.t.Helper()
	if w.sensor == nil {
		w.t.Fatal("No sensor to delete")
	}
	w.t.Log("Deleting sensor", w.sensor.Name)
	ctx := context.Background()
	err := w.sensorClient.Delete(ctx, w.sensor.Name, metav1.DeleteOptions{})
	if err != nil {
		w.t.Fatal(err)
	}
	return w
}

func (w *When) Wait(timeout time.Duration) *When {
	w.t.Helper()
	w.t.Log("Waiting for", timeout.String())
	time.Sleep(timeout)
	w.t.Log("Done waiting")
	return w
}

func (w *When) And(block func()) *When {
	w.t.Helper()
	block()
	if w.t.Failed() {
		w.t.FailNow()
	}
	return w
}

func (w *When) Exec(name string, args []string, block func(t *testing.T, output string, err error)) *When {
	w.t.Helper()
	output, err := Exec(name, args...)
	block(w.t, output, err)
	if w.t.Failed() {
		w.t.FailNow()
	}
	return w
}

func (w *When) WaitForEventBusReady() *When {
	w.t.Helper()
	ctx := context.Background()
	if err := testutil.WaitForEventBusReady(ctx, w.eventBusClient, w.eventBus.Name, defaultTimeout); err != nil {
		w.t.Fatal(err)
	}
	if w.eventBus.Spec.Kafka == nil { // not needed for kafka (exotic only)
		if err := testutil.WaitForEventBusStatefulSetReady(ctx, w.kubeClient, Namespace, w.eventBus.Name, 2*time.Minute); err != nil {
			w.t.Fatal(err)
		}
	}
	return w
}

func (w *When) WaitForEventSourceReady() *When {
	w.t.Helper()
	ctx := context.Background()
	if err := testutil.WaitForEventSourceReady(ctx, w.eventSourceClient, w.eventSource.Name, defaultTimeout); err != nil {
		w.t.Fatal(err)
	}
	if err := testutil.WaitForEventSourceDeploymentReady(ctx, w.kubeClient, Namespace, w.eventSource.Name, defaultTimeout); err != nil {
		w.t.Fatal(err)
	}
	w.t.Logf("Pod of EventSource %s is running", w.eventSource.Name)
	return w
}

func (w *When) WaitForSensorReady() *When {
	w.t.Helper()
	ctx := context.Background()
	if err := testutil.WaitForSensorReady(ctx, w.sensorClient, w.sensor.Name, defaultTimeout); err != nil {
		w.t.Fatal(err)
	}
	if err := testutil.WaitForSensorDeploymentReady(ctx, w.kubeClient, Namespace, w.sensor.Name, defaultTimeout); err != nil {
		w.t.Fatal(err)
	}
	w.t.Logf("Pod of Sensor %s is running", w.sensor.Name)
	return w
}

func (w *When) Given() *Given {
	return &Given{
		t:                 w.t,
		eventBusClient:    w.eventBusClient,
		eventSourceClient: w.eventSourceClient,
		sensorClient:      w.sensorClient,
		eventBus:          w.eventBus,
		eventSource:       w.eventSource,
		sensor:            w.sensor,
		restConfig:        w.restConfig,
		kubeClient:        w.kubeClient,
	}
}

func (w *When) Then() *Then {
	return &Then{
		t:                 w.t,
		eventBusClient:    w.eventBusClient,
		eventSourceClient: w.eventSourceClient,
		sensorClient:      w.sensorClient,
		eventBus:          w.eventBus,
		eventSource:       w.eventSource,
		sensor:            w.sensor,
		restConfig:        w.restConfig,
		kubeClient:        w.kubeClient,
	}
}
