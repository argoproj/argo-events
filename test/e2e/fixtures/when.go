package fixtures

import (
	"context"
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	eventbuspkg "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/typed/eventbus/v1alpha1"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/typed/eventsource/v1alpha1"
	sensorpkg "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
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
	timeout := defaultTimeout
	fieldSelector := "metadata.name=" + w.eventBus.Name
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	watch, err := w.eventBusClient.Watch(ctx, opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer watch.Stop()
	timeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		timeoutCh <- true
	}()
	for {
		select {
		case event := <-watch.ResultChan():
			eb, ok := event.Object.(*eventbusv1alpha1.EventBus)
			if ok {
				if eb.Status.IsReady() {
					w.eventBus = eb
					return w
				}
			} else {
				w.t.Fatal("not eventbus")
			}
		case <-timeoutCh:
			w.t.Fatalf("timeout after %v waiting for EventBus ready", timeout)
		}
	}
}

func (w *When) WaitForEventBusStatefulSetReady() *When {
	w.t.Helper()
	timeout := defaultTimeout
	labelSelector := fmt.Sprintf("controller=eventbus-controller,eventbus-name=%s", w.eventBus.Name)
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	ctx := context.Background()
	watch, err := w.kubeClient.AppsV1().StatefulSets(Namespace).Watch(ctx, opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer watch.Stop()
	timeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		timeoutCh <- true
	}()

statefulSetWatch:
	for {
		select {
		case event := <-watch.ResultChan():
			ss, ok := event.Object.(*appsv1.StatefulSet)
			if ok {
				if ss.Status.Replicas == ss.Status.ReadyReplicas {
					break statefulSetWatch
				}
			} else {
				w.t.Fatal("not statefulset")
			}
		case <-timeoutCh:
			w.t.Fatalf("timeout after %v waiting for EventBus StatefulSet ready", timeout)
		}
	}

	// POD
	podWatch, err := w.kubeClient.CoreV1().Pods(Namespace).Watch(ctx, opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer podWatch.Stop()
	podTimeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		podTimeoutCh <- true
	}()

	podNames := make(map[string]bool)
	for {
		if len(podNames) == 3 {
			// defaults to 3 Pods
			return w
		}
		select {
		case event := <-podWatch.ResultChan():
			p, ok := event.Object.(*corev1.Pod)
			if ok {
				if p.Status.Phase == corev1.PodRunning {
					if _, existing := podNames[p.GetName()]; !existing {
						podNames[p.GetName()] = true
					}
				}
			} else {
				w.t.Fatal("not Pod")
			}
		case <-podTimeoutCh:
			w.t.Fatalf("timeout after %v waiting for event bus Pod ready", timeout)
		}
	}
}

func (w *When) WaitForEventSourceReady() *When {
	w.t.Helper()
	ctx := context.Background()
	timeout := defaultTimeout
	fieldSelector := "metadata.name=" + w.eventSource.Name
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	watch, err := w.eventSourceClient.Watch(ctx, opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer watch.Stop()
	timeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		timeoutCh <- true
	}()
	for {
		select {
		case event := <-watch.ResultChan():
			es, ok := event.Object.(*eventsourcev1alpha1.EventSource)
			if ok {
				if es.Status.IsReady() {
					w.eventSource = es
					return w
				}
			} else {
				w.t.Fatal("not eventsource")
			}
		case <-timeoutCh:
			w.t.Fatalf("timeout after %v waiting for EventSource ready", timeout)
		}
	}
}

func (w *When) WaitForEventSourceDeploymentReady() *When {
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", w.eventSource.Name)
	return w.waitForDeploymentAndPodReady("EventSource", labelSelector, defaultTimeout)
}

func (w *When) WaitForSensorReady() *When {
	w.t.Helper()
	ctx := context.Background()
	timeout := defaultTimeout
	fieldSelector := "metadata.name=" + w.sensor.Name
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	watch, err := w.sensorClient.Watch(ctx, opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer watch.Stop()
	timeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		timeoutCh <- true
	}()
	for {
		select {
		case event := <-watch.ResultChan():
			s, ok := event.Object.(*sensorv1alpha1.Sensor)
			if ok {
				if s.Status.IsReady() {
					w.sensor = s
					return w
				}
			} else {
				w.t.Fatal("not sensor")
			}
		case <-timeoutCh:
			w.t.Fatalf("timeout after %v waiting for Sensor ready", timeout)
		}
	}
}

func (w *When) WaitForSensorDeploymentReady() *When {
	labelSelector := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", w.sensor.Name)
	return w.waitForDeploymentAndPodReady("Sensor", labelSelector, 60*time.Second)
}

func (w *When) waitForDeploymentAndPodReady(objectType, labelSelector string, timeout time.Duration) *When {
	w.t.Helper()
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	ctx := context.Background()
	deployWatch, err := w.kubeClient.AppsV1().Deployments(Namespace).Watch(ctx, opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer deployWatch.Stop()
	deployTimeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		deployTimeoutCh <- true
	}()

deployWatch:
	for {
		select {
		case event := <-deployWatch.ResultChan():
			ss, ok := event.Object.(*appsv1.Deployment)
			if ok {
				if ss.Status.Replicas == ss.Status.AvailableReplicas {
					break deployWatch
				}
			} else {
				w.t.Fatal("not deployment")
			}
		case <-deployTimeoutCh:
			w.t.Fatalf("timeout after %v waiting for %s Deployment ready", timeout, objectType)
		}
	}

	// POD
	podWatch, err := w.kubeClient.CoreV1().Pods(Namespace).Watch(ctx, opts)
	if err != nil {
		w.t.Fatal(err)
	}
	defer podWatch.Stop()
	podTimeoutCh := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		podTimeoutCh <- true
	}()
	for {
		select {
		case event := <-podWatch.ResultChan():
			p, ok := event.Object.(*corev1.Pod)
			if ok {
				if p.Status.Phase == corev1.PodRunning {
					w.t.Logf("%s pod is running", objectType)
					return w
				}
			} else {
				w.t.Fatal("not Pod")
			}
		case <-podTimeoutCh:
			w.t.Fatalf("timeout after %v waiting for %s Pod ready", timeout, objectType)
		}
	}
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
