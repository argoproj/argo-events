package util

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	eventbuspkg "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/typed/eventbus/v1alpha1"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/typed/eventsource/v1alpha1"
	sensorpkg "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
)

func WaitForEventBusReady(ctx context.Context, eventBusClient eventbuspkg.EventBusInterface, eventBusName string, timeout time.Duration) error {
	fieldSelector := "metadata.name=" + eventBusName
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v waiting for EventBus ready", timeout)
		default:
		}
		ebList, err := eventBusClient.List(ctx, opts)
		if err != nil {
			return fmt.Errorf("error getting EventBus list: %w", err)
		}
		if len(ebList.Items) > 0 && ebList.Items[0].Status.IsReady() {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func WaitForEventBusStatefulSetReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, eventBusName string, timeout time.Duration) error {
	labelSelector := fmt.Sprintf("controller=eventbus-controller,eventbus-name=%s", eventBusName)
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v waiting for EventBus StatefulSet ready", timeout)
		default:
		}
		stsList, err := kubeClient.AppsV1().StatefulSets(namespace).List(ctx, opts)
		if err != nil {
			return fmt.Errorf("error getting EventBus StatefulSet list: %w", err)
		}
		if len(stsList.Items) > 0 && stsList.Items[0].Status.Replicas == stsList.Items[0].Status.ReadyReplicas {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func WaitForEventSourceReady(ctx context.Context, eventSourceClient eventsourcepkg.EventSourceInterface, eventSourceName string, timeout time.Duration) error {
	fieldSelector := "metadata.name=" + eventSourceName
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v waiting for EventSource ready", timeout)
		default:
		}
		esList, err := eventSourceClient.List(ctx, opts)
		if err != nil {
			return fmt.Errorf("error getting EventSource list: %w", err)
		}
		if len(esList.Items) > 0 && esList.Items[0].Status.IsReady() {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func WaitForEventSourceDeploymentReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, eventSourceName string, timeout time.Duration) error {
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", eventSourceName)
	return waitForDeploymentAndPodReady(ctx, kubeClient, namespace, "EventSource", labelSelector, timeout)
}

func WaitForSensorReady(ctx context.Context, sensorClient sensorpkg.SensorInterface, sensorName string, timeout time.Duration) error {
	fieldSelector := "metadata.name=" + sensorName
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v waiting for Sensor ready", timeout)
		default:
		}
		sensorList, err := sensorClient.List(ctx, opts)
		if err != nil {
			return fmt.Errorf("error getting Sensor list: %w", err)
		}
		if len(sensorList.Items) > 0 && sensorList.Items[0].Status.IsReady() {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func WaitForSensorDeploymentReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, sensorName string, timeout time.Duration) error {
	labelSelector := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", sensorName)
	return waitForDeploymentAndPodReady(ctx, kubeClient, namespace, "Sensor", labelSelector, timeout)
}

func waitForDeploymentAndPodReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, objectType, labelSelector string, timeout time.Duration) error {
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v waiting for deployment ready", timeout)
		default:
		}
		deployList, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, opts)
		if err != nil {
			return fmt.Errorf("error getting deployment list: %w", err)
		}
		ok := len(deployList.Items) == 1
		if !ok {
			continue
		}
		ok = ok && deployList.Items[0].Status.Replicas == deployList.Items[0].Status.ReadyReplicas
		podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
		if err != nil {
			return fmt.Errorf("error getting deployment pod list: %w", err)
		}
		ok = ok && len(podList.Items) > 0 && len(podList.Items) == int(*deployList.Items[0].Spec.Replicas)
		for _, p := range podList.Items {
			ok = ok && p.Status.Phase == corev1.PodRunning
		}
		if ok {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

type podLogCheckOptions struct {
	timeout time.Duration
	count   int
}

func defaultPodLogCheckOptions() *podLogCheckOptions {
	return &podLogCheckOptions{
		timeout: 15 * time.Second,
		count:   -1,
	}
}

type PodLogCheckOption func(*podLogCheckOptions)

func PodLogCheckOptionWithTimeout(t time.Duration) PodLogCheckOption {
	return func(o *podLogCheckOptions) {
		o.timeout = t
	}
}

func PodLogCheckOptionWithCount(c int) PodLogCheckOption {
	return func(o *podLogCheckOptions) {
		o.count = c
	}
}

func EventSourcePodLogContains(ctx context.Context, kubeClient kubernetes.Interface, namespace, eventSourceName, regex string, options ...PodLogCheckOption) (bool, error) {
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", eventSourceName)
	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		return false, fmt.Errorf("error getting event source pod name: %w", err)
	}

	return PodsLogContains(ctx, kubeClient, namespace, regex, podList, options...), nil
}

func SensorPodLogContains(ctx context.Context, kubeClient kubernetes.Interface, namespace, sensorName, regex string, options ...PodLogCheckOption) (bool, error) {
	labelSelector := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", sensorName)
	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		return false, fmt.Errorf("error getting sensor pod name: %w", err)
	}

	return PodsLogContains(ctx, kubeClient, namespace, regex, podList, options...), nil
}

func PodsLogContains(ctx context.Context, kubeClient kubernetes.Interface, namespace, regex string, podList *corev1.PodList, options ...PodLogCheckOption) bool {
	// parse options
	o := defaultPodLogCheckOptions()
	for _, opt := range options {
		if opt != nil {
			opt(o)
		}
	}

	cctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	errChan := make(chan error)
	resultChan := make(chan bool)
	wg := &sync.WaitGroup{}
	for _, p := range podList.Items {
		wg.Add(1)
		go func(podName string) {
			defer wg.Done()
			fmt.Printf("Watching POD: %s\n", podName)
			var contains bool
			var err error
			if o.count == -1 {
				contains, err = podLogContains(cctx, kubeClient, namespace, podName, regex)
			} else {
				contains, err = podLogContainsCount(cctx, kubeClient, namespace, podName, regex, o.count)
			}
			if err != nil {
				errChan <- err
				return
			}
			if contains {
				resultChan <- true
			}
		}(p.Name)
	}
	allDone := make(chan bool)
	go func() {
		wg.Wait()
		close(allDone)
	}()
	for {
		select {
		case result := <-resultChan:
			if result {
				return true
			} else {
				fmt.Println("read resultChan but not true")
			}
		case err := <-errChan:
			fmt.Printf("error: %v", err)
		case <-allDone:
			for len(resultChan) > 0 {
				if x := <-resultChan; x {
					return true
				}
			}
			return false
		}
	}
}

// look for at least one instance of the regex string in the log
func podLogContains(ctx context.Context, client kubernetes.Interface, namespace, podName, regex string) (bool, error) {
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
			fmt.Println(string(data))
			if exp.Match(data) {
				return true, nil
			}
		}
	}
}

// look for a specific number of instances of the regex string in the log
func podLogContainsCount(ctx context.Context, client kubernetes.Interface, namespace, podName, regex string, count int) (bool, error) {
	stream, err := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Follow: true}).Stream(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = stream.Close() }()

	exp, err := regexp.Compile(regex)
	if err != nil {
		return false, err
	}

	instancesChan := make(chan struct{})

	// scan the log looking for matches
	go func(ctx context.Context, instancesChan chan<- struct{}) {
		s := bufio.NewScanner(stream)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if !s.Scan() {
					return
				}
				data := s.Bytes()
				fmt.Println(string(data))
				if exp.Match(data) {
					instancesChan <- struct{}{}
				}
			}
		}
	}(ctx, instancesChan)

	actualCount := 0
	for {
		select {
		case <-instancesChan:
			actualCount++
		case <-ctx.Done():
			fmt.Printf("time:%v, count:%d,actualCount:%d\n", time.Now().Unix(), count, actualCount)
			return count == actualCount, nil
		}
	}
}

func WaitForNoPodFound(ctx context.Context, kubeClient kubernetes.Interface, namespace, labelSelector string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return fmt.Errorf("error getting pod list: %w", err)
		}
		if len(podList.Items) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pod disappearing")
		default:
		}
		time.Sleep(2 * time.Second)
	}
}
