package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	eventbusversiond "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned"
	eventbuspkg "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/typed/eventbus/v1alpha1"
	eventsourceversiond "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/typed/eventsource/v1alpha1"
	sensorversiond "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	sensorpkg "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
	testutil "github.com/argoproj/argo-events/test/util"
)

const (
	eventBusName   = "stress-test"
	defaultTimeout = 60 * time.Second

	logEventSourceStarted      = "Eventing server started."
	logSensorStarted           = "Sensor started."
	logTriggerActionSuccessful = "successfully processed the trigger"
	logTriggerActionFailed     = "failed to trigger action"
)

var (
	eventBus = `apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native:
      auth: token`

	webhookEventSource = `apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: stress-test-webhook
spec:
  webhook:
    test:
      port: "12000"
      endpoint: /test
      method: POST`

	webhookSensor = `apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: stress-test-log
spec:
  dependencies:
  - name: test-dep
    eventSourceName: stress-test-webhook
    eventName: test
  triggers:
  - template:
      name: log-trigger
      log: {}`
)

func createEventBus(ctx context.Context, kubeClient kubernetes.Interface, eventBusClient eventbuspkg.EventBusInterface, namespace string) (*eventbusv1alpha1.EventBus, error) {
	eb := &eventbusv1alpha1.EventBus{}
	if err := yaml.Unmarshal([]byte(eventBus), eb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event bus yaml: %w", err)
	}
	eb.Name = eventBusName
	result, err := eventBusClient.Create(ctx, eb, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create event bus: %w", err)
	}
	if err := testutil.WaitForEventBusReady(ctx, eventBusClient, eb.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see event bus ready: %w", err)
	}
	if err := testutil.WaitForEventBusStatefulSetReady(ctx, kubeClient, namespace, eb.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see event bus statefulset ready: %w", err)
	}
	return result, nil
}

func createEventSource(ctx context.Context, kubeClient kubernetes.Interface, eventSourceClient eventsourcepkg.EventSourceInterface, namespace string) (*eventsourcev1alpha1.EventSource, error) {
	es := &eventsourcev1alpha1.EventSource{}
	if err := yaml.Unmarshal([]byte(webhookEventSource), es); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event source yaml: %w", err)
	}
	es.Spec.EventBusName = eventBusName
	result, err := eventSourceClient.Create(ctx, es, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create event source: %w", err)
	}
	if err := testutil.WaitForEventSourceReady(ctx, eventSourceClient, es.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see event source ready: %w", err)
	}
	if err := testutil.WaitForEventSourceDeploymentReady(ctx, kubeClient, namespace, es.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see event source deployment and pod ready: %w", err)
	}
	contains, err := testutil.EventSourcePodLogContains(ctx, kubeClient, namespace, es.Name, logEventSourceStarted, defaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("expected to see event source pod contains something: %w", err)
	}
	if !contains {
		return nil, fmt.Errorf("EventSource Pod does look good, it might have started failed")
	}
	return result, nil
}

func createSensor(ctx context.Context, kubeClient kubernetes.Interface, sensorClient sensorpkg.SensorInterface, namespace string) (*sensorv1alpha1.Sensor, error) {
	sensor := &sensorv1alpha1.Sensor{}
	if err := yaml.Unmarshal([]byte(webhookSensor), sensor); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sensor yaml: %w", err)
	}
	sensor.Spec.EventBusName = eventBusName
	result, err := sensorClient.Create(ctx, sensor, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create sensor: %w", err)
	}
	if err := testutil.WaitForSensorReady(ctx, sensorClient, sensor.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see sensor ready: %w", err)
	}
	if err := testutil.WaitForSensorDeploymentReady(ctx, kubeClient, namespace, sensor.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see sensor deployment and pod ready: %w", err)
	}
	contains, err := testutil.SensorPodLogContains(ctx, kubeClient, namespace, sensor.Name, logSensorStarted, defaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("expected to see sensor pod contains something: %w", err)
	}
	if !contains {
		return nil, fmt.Errorf("Sensor Pod does look good, it might have started failed")
	}
	return result, nil
}

func runTesting(ctx context.Context, kubeClient kubernetes.Interface, namespace, sensorPodName string, num int, timeout time.Duration) {
	succeededSent := 0
	failedSent := 0
	finishedSent := false

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		stream, err := kubeClient.CoreV1().Pods(namespace).GetLogs(sensorPodName, &corev1.PodLogOptions{Follow: true}).Stream(ctx)
		if err != nil {
			fmt.Printf("failed to acquire sensor pod log stream: %v", err)
			return
		}
		defer func() { _ = stream.Close() }()

		successReg, err := regexp.Compile(logTriggerActionSuccessful)
		if err != nil {
			fmt.Printf("failed to compile regex for success pattern: %v", err)
			return
		}
		failureReg, err := regexp.Compile(logTriggerActionFailed)
		if err != nil {
			fmt.Printf("failed to compile regex for failure pattern: %v", err)
			return
		}

		succeeded := 0
		failed := 0

		var firstActionTime time.Time
		var lastActionTime time.Time

		defer func() {
			fmt.Printf("\n++++++++++++++++++ Action Triggered Summary ++++++++++++++++++\n")
			fmt.Printf("Expected actions              : %d\n", num)
			fmt.Printf("Action triggered successfully : %d\n", succeeded)
			fmt.Printf("Action triggered failed       : %d\n", failed)
			fmt.Printf("First action triggered at     : %v\n", firstActionTime)
			fmt.Printf("Last action triggered at      : %v\n", lastActionTime)
			fmt.Printf("Total time taken              : %v\n", time.Since(firstActionTime))
		}()

		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		s := bufio.NewScanner(stream)
		for {
			select {
			case <-cctx.Done():
				return
			default:
				if !s.Scan() {
					fmt.Printf("error found: %v", s.Err())
					return
				}
				data := s.Bytes()
				if successReg.Match(data) {
					succeeded++
				} else if failureReg.Match(data) {
					failed++
				}
				if succeeded+failed == 1 {
					firstActionTime = time.Now()
				}
				lastActionTime = time.Now()
				if finishedSent && succeededSent > 0 && succeeded+failed >= succeededSent {
					return
				}
			}
		}
	}()

	startTime := time.Now()

	for i := 0; i < num; i++ {
		resp, err := http.Post("http://localhost:12000/test", "application/json", bytes.NewBuffer([]byte("{}")))
		if err != nil {
			fmt.Printf("failed to post events: %v", err)
			return
		}
		if resp.StatusCode == 200 {
			succeededSent++
		} else {
			failedSent++
		}
	}
	finishedSent = true
	fmt.Printf("\n++++++++++++++++++++  Events Sent Summary ++++++++++++++++++++ \n")
	fmt.Printf("Total requests                : %d\n", num)
	fmt.Printf("Succeeded                     : %d\n", succeededSent)
	fmt.Printf("Failed                        : %d\n", failedSent)
	fmt.Printf("First event sent at           : %v\n", startTime)
	fmt.Printf("Last event sent at            : %v\n", time.Now())
	fmt.Printf("Total time taken              : %v\n", time.Since(startTime))
	wg.Wait()
}

func mainFunc(num int, timeout time.Duration) error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return err
	}
	namespace, _, _ := kubeConfig.Namespace()
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	eventBusClient := eventbusversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().EventBus(namespace)
	eventSourceClient := eventsourceversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().EventSources(namespace)
	sensorClient := sensorversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().Sensors(namespace)

	fmt.Println("################## Preparing ##################")
	ctx := context.Background()

	// Create EventBus
	eb, err := createEventBus(ctx, kubeClient, eventBusClient, namespace)
	if err != nil {
		return err
	}
	defer func() {
		_ = eventBusClient.Delete(ctx, eb.Name, metav1.DeleteOptions{})
	}()

	time.Sleep(5 * time.Second)

	// Create Event Source
	es, err := createEventSource(ctx, kubeClient, eventSourceClient, namespace)
	if err != nil {
		return err
	}
	defer func() {
		_ = eventSourceClient.Delete(ctx, es.Name, metav1.DeleteOptions{})
	}()

	// Create Sensor
	sensor, err := createSensor(ctx, kubeClient, sensorClient, namespace)
	if err != nil {
		return err
	}
	defer func() {
		_ = sensorClient.Delete(ctx, sensor.Name, metav1.DeleteOptions{})
	}()

	// Port-forward
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", es.Name)
	esPodList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		return err
	}
	esPodName := esPodList.Items[0].GetName()
	fmt.Printf("EventSource POD name: %s\n", esPodName)

	stopCh := make(chan struct{}, 1)
	if err = testutil.PodPortForward(config, namespace, esPodName, 12000, 12000, stopCh); err != nil {
		return err
	}
	defer close(stopCh)

	// Run testing
	fmt.Println("")
	fmt.Println("################## Start Testing ##################")
	fmt.Println("")

	ls := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", sensor.Name)
	sensorPodList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: ls, FieldSelector: "status.phase=Running"})
	if err != nil {
		return err
	}
	sensorPodName := sensorPodList.Items[0].GetName()

	time.Sleep(3 * time.Second)

	runTesting(ctx, kubeClient, namespace, sensorPodName, num, timeout)

	time.Sleep(20 * time.Second)

	return nil
}

func main() {
	num := 100
	timeout := 60 * time.Second

	if err := mainFunc(num, timeout); err != nil {
		panic(err)
	}
}
