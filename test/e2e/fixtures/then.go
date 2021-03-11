package fixtures

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	eventbuspkg "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/typed/eventbus/v1alpha1"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/typed/eventsource/v1alpha1"
	sensorpkg "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Then struct {
	t                 *testing.T
	eventBusClient    eventbuspkg.EventBusInterface
	eventSourceClient eventsourcepkg.EventSourceInterface
	sensorClient      sensorpkg.SensorInterface
	eventBus          *eventbusv1alpha1.EventBus
	eventSource       *eventsourcev1alpha1.EventSource
	sensor            *sensorv1alpha1.Sensor
	restConfig        *rest.Config
	kubeClient        kubernetes.Interface

	portForwarderStopChanels map[string]chan struct{}
}

func (t *Then) ExpectEventBusDeleted() *Then {
	ctx := context.Background()
	_, err := t.eventBusClient.Get(ctx, t.eventBus.Name, metav1.GetOptions{})
	if err == nil || !apierr.IsNotFound(err) {
		t.t.Fatalf("expected event bus to be deleted: %v", err)
	}
	return t
}

func (t *Then) ExpectEventSourcePodLogContains(regex string) *Then {
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", t.eventSource.Name)
	ctx := context.Background()
	podList, err := t.kubeClient.CoreV1().Pods(Namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		t.t.Fatalf("error getting event source pod name: %v", err)
	}
	podName := podList.Items[0].GetName()
	t.t.Logf("EventSource POD name: %s", podName)
	timeout := defaultTimeout
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	contains, err := t.podLogContains(cctx, t.kubeClient, Namespace, podName, regex)
	if err != nil {
		t.t.Fatalf("expected event source pod logs: %v", err)
	}
	if !contains {
		t.t.Fatalf("expected event source pod log contains %s", regex)
	}
	return t
}

func (t *Then) ExpectSensorPodLogContains(regex string) *Then {
	labelSelector := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", t.sensor.Name)
	ctx := context.Background()
	podList, err := t.kubeClient.CoreV1().Pods(Namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		t.t.Fatalf("error getting event source pod name: %v", err)
	}
	podName := podList.Items[0].GetName()
	t.t.Logf("Sensor POD name: %s", podName)
	timeout := defaultTimeout
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	contains, err := t.podLogContains(cctx, t.kubeClient, Namespace, podName, regex)
	if err != nil {
		t.t.Fatalf("expected sensor pod logs: %v", err)
	}
	if !contains {
		t.t.Fatalf("expected sensor pod log contains %s", regex)
	}
	return t
}

func (t *Then) podLogContains(ctx context.Context, client kubernetes.Interface, namespace, podName, regex string) (bool, error) {
	stream, err := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Follow: true}).Stream(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = stream.Close() }()

	exp, err := regexp.Compile(regex)
	if err != nil {
		return false, err
	}

	s := bufio.NewScanner(stream)
	for {
		select {
		case <-ctx.Done():
			return false, nil
		default:
			if !s.Scan() {
				return false, s.Err()
			}
			data := s.Bytes()
			t.t.Log(string(data))
			if exp.Match(data) {
				return true, nil
			}
		}
	}
}

func (t *Then) EventSourcePodPortForward(localPort, remotePort int) *Then {
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", t.eventSource.Name)
	ctx := context.Background()
	podList, err := t.kubeClient.CoreV1().Pods(Namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		t.t.Fatalf("error getting event source pod name: %v", err)
	}
	podName := podList.Items[0].GetName()
	t.t.Logf("EventSource POD name: %s", podName)

	stopCh := make(chan struct{}, 1)
	if err = PortForward(t.restConfig, Namespace, podName, localPort, remotePort, stopCh); err != nil {
		t.t.Fatalf("expected eventsource pod port-forward: %v", err)
	}
	if t.portForwarderStopChanels == nil {
		t.portForwarderStopChanels = make(map[string]chan struct{})
	}
	t.portForwarderStopChanels[podName] = stopCh
	return t
}

func (t *Then) SensorPodPortForward(localPort, remotePort int) *Then {
	labelSelector := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", t.sensor.Name)
	ctx := context.Background()
	podList, err := t.kubeClient.CoreV1().Pods(Namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		t.t.Fatalf("error getting sensor pod name: %v", err)
	}
	podName := podList.Items[0].GetName()
	t.t.Logf("Sensor POD name: %s", podName)

	stopCh := make(chan struct{}, 1)
	if err = PortForward(t.restConfig, Namespace, podName, localPort, remotePort, stopCh); err != nil {
		t.t.Fatalf("expected sensor pod port-forward: %v", err)
	}
	if t.portForwarderStopChanels == nil {
		t.portForwarderStopChanels = make(map[string]chan struct{})
	}
	t.portForwarderStopChanels[podName] = stopCh
	return t
}

func (t *Then) TerminateAllPodPortForwards() *Then {
	if len(t.portForwarderStopChanels) > 0 {
		for k, v := range t.portForwarderStopChanels {
			t.t.Logf("Terminating port-forward for POD %s", k)
			close(v)
		}
	}
	return t
}

func (t *Then) When() *When {
	return &When{
		t:                 t.t,
		eventBusClient:    t.eventBusClient,
		eventSourceClient: t.eventSourceClient,
		sensorClient:      t.sensorClient,
		eventBus:          t.eventBus,
		eventSource:       t.eventSource,
		sensor:            t.sensor,
		restConfig:        t.restConfig,
		kubeClient:        t.kubeClient,
	}
}
