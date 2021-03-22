package util

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	eventbuspkg "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/typed/eventbus/v1alpha1"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/typed/eventsource/v1alpha1"
	sensorpkg "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
)

func WaitForEventBusReady(ctx context.Context, eventBusClient eventbuspkg.EventBusInterface, eventBusName string, timeout time.Duration) error {
	fieldSelector := "metadata.name=" + eventBusName
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	watch, err := eventBusClient.Watch(ctx, opts)
	if err != nil {
		return err
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
					return nil
				}
			} else {
				return fmt.Errorf("not eventbus")
			}
		case <-timeoutCh:
			return fmt.Errorf("timeout after %v waiting for EventBus ready", timeout)
		}
	}
}

func WaitForEventBusStatefulSetReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, eventBusName string, timeout time.Duration) error {
	labelSelector := fmt.Sprintf("controller=eventbus-controller,eventbus-name=%s", eventBusName)
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	watch, err := kubeClient.AppsV1().StatefulSets(namespace).Watch(ctx, opts)
	if err != nil {
		return err
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
				return fmt.Errorf("not statefulset")
			}
		case <-timeoutCh:
			return fmt.Errorf("timeout after %v waiting for EventBus StatefulSet ready", timeout)
		}
	}

	// POD
	podWatch, err := kubeClient.CoreV1().Pods(namespace).Watch(ctx, opts)
	if err != nil {
		return err
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
			return nil
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
				return fmt.Errorf("not pod")
			}
		case <-podTimeoutCh:
			return fmt.Errorf("timeout after %v waiting for event bus Pod ready", timeout)
		}
	}
}

func WaitForEventSourceReady(ctx context.Context, eventSourceClient eventsourcepkg.EventSourceInterface, eventSourceName string, timeout time.Duration) error {
	fieldSelector := "metadata.name=" + eventSourceName
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	watch, err := eventSourceClient.Watch(ctx, opts)
	if err != nil {
		return err
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
					return nil
				}
			} else {
				return fmt.Errorf("not eventsource")
			}
		case <-timeoutCh:
			return fmt.Errorf("timeout after %v waiting for EventSource ready", timeout)
		}
	}
}

func WaitForEventSourceDeploymentReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, eventSourceName string, timeout time.Duration) error {
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", eventSourceName)
	return waitForDeploymentAndPodReady(ctx, kubeClient, namespace, "EventSource", labelSelector, timeout)
}

func WaitForSensorReady(ctx context.Context, sensorClient sensorpkg.SensorInterface, sensorName string, timeout time.Duration) error {
	fieldSelector := "metadata.name=" + sensorName
	opts := metav1.ListOptions{FieldSelector: fieldSelector}
	watch, err := sensorClient.Watch(ctx, opts)
	if err != nil {
		return err
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
					return nil
				}
			} else {
				return fmt.Errorf("not sensor")
			}
		case <-timeoutCh:
			return fmt.Errorf("timeout after %v waiting for Sensor ready", timeout)
		}
	}
}

func WaitForSensorDeploymentReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, sensorName string, timeout time.Duration) error {
	labelSelector := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", sensorName)
	return waitForDeploymentAndPodReady(ctx, kubeClient, namespace, "Sensor", labelSelector, timeout)
}

func waitForDeploymentAndPodReady(ctx context.Context, kubeClient kubernetes.Interface, namespace, objectType, labelSelector string, timeout time.Duration) error {
	opts := metav1.ListOptions{LabelSelector: labelSelector}
	deployWatch, err := kubeClient.AppsV1().Deployments(namespace).Watch(ctx, opts)
	if err != nil {
		return err
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
				return fmt.Errorf("not deployment")
			}
		case <-deployTimeoutCh:
			return fmt.Errorf("timeout after %v waiting for %s Deployment ready", timeout, objectType)
		}
	}

	// POD
	podWatch, err := kubeClient.CoreV1().Pods(namespace).Watch(ctx, opts)
	if err != nil {
		return err
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
					return nil
				}
			} else {
				return fmt.Errorf("not Pod")
			}
		case <-podTimeoutCh:
			return fmt.Errorf("timeout after %v waiting for %s Pod ready", timeout, objectType)
		}
	}
}

func EventSourcePodLogContains(ctx context.Context, kubeClient kubernetes.Interface, namespace, eventSourceName, regex string, timeout time.Duration) (bool, error) {
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", eventSourceName)
	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		return false, fmt.Errorf("error getting event source pod name: %w", err)
	}
	podName := podList.Items[0].GetName()
	fmt.Printf("EventSource POD name: %s\n", podName)
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return podLogContains(cctx, kubeClient, namespace, podName, regex)
}

func SensorPodLogContains(ctx context.Context, kubeClient kubernetes.Interface, namespace, sensorName, regex string, timeout time.Duration) (bool, error) {
	labelSelector := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", sensorName)
	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		return false, fmt.Errorf("error getting sensor pod name: %w", err)
	}
	podName := podList.Items[0].GetName()
	fmt.Printf("Sensor POD name: %s", podName)
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return podLogContains(cctx, kubeClient, namespace, podName, regex)
}

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
