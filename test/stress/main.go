package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsversiond "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	eventspkg "github.com/argoproj/argo-events/pkg/client/clientset/versioned/typed/events/v1alpha1"
	testutil "github.com/argoproj/argo-events/test/util"
)

const (
	eventBusName   = "stress-testing"
	defaultTimeout = 60 * time.Second

	Success = "success"
	Failure = "failure"

	First = "first"
	Last  = "last"

	EventNameKey   = "eventName"
	TriggerNameKey = "triggerName"

	StressTestingLabel      = "argo-events-stress"
	StressTestingLabelValue = "true"

	logEventSourceStarted      = "Eventing server started."
	logSensorStarted           = "Sensor started."
	logTriggerActionSuccessful = "Successfully processed trigger"
	logTriggerActionFailed     = "Failed to execute a trigger"
	logEventSuccessful         = "Succeeded to publish an event"
	logEventFailed             = "Failed to publish an event"
)

type TestingEventSource string

// possible values of TestingEventSource
const (
	UnsupportedEventsource TestingEventSource = "unsupported"
	WebhookEventSource     TestingEventSource = "webhook"
	SQSEventSource         TestingEventSource = "sqs"
	SNSEventSource         TestingEventSource = "sns"
	KafkaEventSource       TestingEventSource = "kafka"
	NATSEventSource        TestingEventSource = "nats"
	RedisEventSource       TestingEventSource = "redis"
)

type EventBusType string

// possible value of EventBus type
const (
	UnsupportedEventBusType EventBusType = "unsupported"
	STANEventBus            EventBusType = "stan"
	JetstreamEventBus       EventBusType = "jetstream"
)

type TestingTrigger string

// possible values of TestingTrigger
const (
	UnsupportedTrigger TestingTrigger = "unsupported"
	WorkflowTrigger    TestingTrigger = "workflow"
	LogTrigger         TestingTrigger = "log"
)

var (
	background = metav1.DeletePropagationBackground
)

type options struct {
	namespace          string
	testingEventSource TestingEventSource
	testingTrigger     TestingTrigger
	eventBusType       EventBusType
	esName             string
	sensorName         string
	// Inactive time before exiting
	idleTimeout time.Duration
	hardTimeout *time.Duration
	noCleanUp   bool

	kubeClient        kubernetes.Interface
	eventBusClient    eventspkg.EventBusInterface
	eventSourceClient eventspkg.EventSourceInterface
	sensorClient      eventspkg.SensorInterface
	restConfig        *rest.Config
}

func NewOptions(testingEventSource TestingEventSource, testingTrigger TestingTrigger, eventBusType EventBusType, esName, sensorName string, idleTimeout time.Duration, noCleanUp bool) (*options, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	namespace, _, _ := kubeConfig.Namespace()
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	eventBusClient := eventsversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().EventBus(namespace)
	eventSourceClient := eventsversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().EventSources(namespace)
	sensorClient := eventsversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().Sensors(namespace)
	return &options{
		namespace:          namespace,
		testingEventSource: testingEventSource,
		testingTrigger:     testingTrigger,
		eventBusType:       eventBusType,
		esName:             esName,
		sensorName:         sensorName,
		kubeClient:         kubeClient,
		eventBusClient:     eventBusClient,
		eventSourceClient:  eventSourceClient,
		restConfig:         config,
		sensorClient:       sensorClient,
		idleTimeout:        idleTimeout,
		noCleanUp:          noCleanUp,
	}, nil
}

func (o *options) createEventBus(ctx context.Context) (*v1alpha1.EventBus, error) {
	fmt.Printf("------- Creating %v EventBus -------\n", o.eventBusType)
	eb := &v1alpha1.EventBus{}
	if err := readResource(fmt.Sprintf("@testdata/eventbus/%v.yaml", o.eventBusType), eb); err != nil {
		return nil, fmt.Errorf("failed to read %v event bus yaml file: %w", o.eventBusType, err)
	}
	l := eb.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[StressTestingLabel] = StressTestingLabelValue
	eb.SetLabels(l)
	eb.Name = eventBusName
	result, err := o.eventBusClient.Create(ctx, eb, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create event bus: %w", err)
	}
	if err := testutil.WaitForEventBusReady(ctx, o.eventBusClient, eb.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see event bus ready: %w", err)
	}
	if err := testutil.WaitForEventBusStatefulSetReady(ctx, o.kubeClient, o.namespace, eb.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see event bus statefulset ready: %w", err)
	}
	return result, nil
}

func (o *options) createEventSource(ctx context.Context) (*v1alpha1.EventSource, error) {
	fmt.Printf("\n------- Creating %v EventSource -------\n", o.testingEventSource)
	es := &v1alpha1.EventSource{}
	file := fmt.Sprintf("@testdata/eventsources/%v.yaml", o.testingEventSource)
	if err := readResource(file, es); err != nil {
		return nil, fmt.Errorf("failed to read %v event source yaml file: %w", o.testingEventSource, err)
	}
	l := es.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[StressTestingLabel] = StressTestingLabelValue
	es.SetLabels(l)
	es.Spec.EventBusName = eventBusName
	result, err := o.eventSourceClient.Create(ctx, es, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create event source: %w", err)
	}
	if err := testutil.WaitForEventSourceReady(ctx, o.eventSourceClient, es.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see event source ready: %w", err)
	}
	if err := testutil.WaitForEventSourceDeploymentReady(ctx, o.kubeClient, o.namespace, es.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see event source deployment and pod ready: %w", err)
	}
	contains, err := testutil.EventSourcePodLogContains(ctx, o.kubeClient, o.namespace, es.Name, logEventSourceStarted)
	if err != nil {
		return nil, fmt.Errorf("expected to see event source pod contains something: %w", err)
	}
	if !contains {
		return nil, fmt.Errorf("EventSource Pod does look good, it might have started failed")
	}
	return result, nil
}

func (o *options) createSensor(ctx context.Context) (*v1alpha1.Sensor, error) {
	fmt.Printf("\n------- Creating %v Sensor -------\n", o.testingTrigger)
	sensor := &v1alpha1.Sensor{}
	file := fmt.Sprintf("@testdata/sensors/%v.yaml", o.testingTrigger)
	if err := readResource(file, sensor); err != nil {
		return nil, fmt.Errorf("failed to read %v sensor yaml file: %w", o.testingTrigger, err)
	}
	l := sensor.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[StressTestingLabel] = StressTestingLabelValue
	sensor.SetLabels(l)
	sensor.Spec.EventBusName = eventBusName
	result, err := o.sensorClient.Create(ctx, sensor, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create sensor: %w", err)
	}
	if err := testutil.WaitForSensorReady(ctx, o.sensorClient, sensor.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see sensor ready: %w", err)
	}
	if err := testutil.WaitForSensorDeploymentReady(ctx, o.kubeClient, o.namespace, sensor.Name, defaultTimeout); err != nil {
		return nil, fmt.Errorf("expected to see sensor deployment and pod ready: %w", err)
	}
	contains, err := testutil.SensorPodLogContains(ctx, o.kubeClient, o.namespace, sensor.Name, logSensorStarted)
	if err != nil {
		return nil, fmt.Errorf("expected to see sensor pod contains something: %w", err)
	}
	if !contains {
		return nil, fmt.Errorf("sensor Pod does look good, it might have started failed")
	}
	return result, nil
}

func (o *options) getEventSourcePodNames(ctx context.Context, eventSourceName string) ([]string, error) {
	labelSelector := fmt.Sprintf("controller=eventsource-controller,eventsource-name=%s", eventSourceName)
	esPodList, err := o.kubeClient.CoreV1().Pods(o.namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		return nil, err
	}
	results := []string{}
	for _, i := range esPodList.Items {
		results = append(results, i.Name)
	}
	return results, nil
}

func (o *options) getSensorPodNames(ctx context.Context, sensorName string) ([]string, error) {
	labelSelector := fmt.Sprintf("controller=sensor-controller,sensor-name=%s", sensorName)
	sPodList, err := o.kubeClient.CoreV1().Pods(o.namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: "status.phase=Running"})
	if err != nil {
		return nil, err
	}
	results := []string{}
	for _, i := range sPodList.Items {
		results = append(results, i.Name)
	}
	return results, nil
}

func (o *options) runTesting(ctx context.Context, eventSourceName, sensorName string) error {
	esPodNames, err := o.getEventSourcePodNames(ctx, eventSourceName)
	if err != nil {
		return fmt.Errorf("failed to get event source pod names: %v", err)
	}
	if len(esPodNames) == 0 {
		return fmt.Errorf("no pod found for event source %s", eventSourceName)
	}
	sensorPodNames, err := o.getSensorPodNames(ctx, sensorName)
	if err != nil {
		return fmt.Errorf("failed to get sensor pod names: %v", err)
	}
	if len(sensorPodNames) == 0 {
		return fmt.Errorf("no pod found for sensor %s", sensorName)
	}

	successActionReg := regexp.MustCompile(logTriggerActionSuccessful)
	failureActionReg := regexp.MustCompile(logTriggerActionFailed)
	successEventReg := regexp.MustCompile(logEventSuccessful)
	failureEventReg := regexp.MustCompile(logEventFailed)

	fmt.Printf(`
*********************************************************
The application will automatically exit:
  - If there's no active events and actions in %v.
`, o.idleTimeout)
	if o.hardTimeout != nil {
		fmt.Printf("  - In %v after it starts.\n", *o.hardTimeout)
	}
	fmt.Printf(`
Or you can terminate it any time by Ctrl + C.
*********************************************************

`)

	esMap := map[string]int64{}
	esTimeMap := map[string]time.Time{}

	sensorMap := map[string]int64{}
	sensorTimeMap := map[string]time.Time{}

	var esLock = &sync.RWMutex{}
	var sensorLock = &sync.RWMutex{}

	startTime := time.Now()

	wg := &sync.WaitGroup{}
	for _, sensorPodName := range sensorPodNames {
		wg.Add(1)
		go func(podName string) {
			defer wg.Done()
			fmt.Printf("Started watching Sensor Pod %s ...\n", podName)
			stream, err := o.kubeClient.CoreV1().Pods(o.namespace).GetLogs(podName, &corev1.PodLogOptions{Follow: true}).Stream(ctx)
			if err != nil {
				fmt.Printf("failed to acquire sensor pod %s log stream: %v", podName, err)
				return
			}
			defer func() { _ = stream.Close() }()

			sCh := make(chan string)
			go func(dataCh chan string) {
				s := bufio.NewScanner(stream)
				for {
					if !s.Scan() {
						fmt.Printf("Can not read: %v\n", s.Err())
						close(dataCh)
						return
					}
					data := s.Bytes()
					triggerName := getLogValue(data, startTime, TriggerNameKey)
					if triggerName == "" {
						continue
					}
					if successActionReg.Match(data) {
						dataCh <- triggerName + "/" + Success
					} else if failureActionReg.Match(data) {
						dataCh <- triggerName + "/" + Failure
					}
				}
			}(sCh)

			for {
				if o.hardTimeout != nil && time.Since(startTime).Seconds() > o.hardTimeout.Seconds() {
					fmt.Printf("Exited Sensor Pod %s due to the hard timeout %v\n", podName, *o.hardTimeout)
					return
				}
				timeout := 5 * 60 * time.Second
				lastActionTime := startTime
				sensorLock.RLock()
				if len(sensorMap) > 0 && len(sensorTimeMap) > 0 {
					timeout = o.idleTimeout
					for _, v := range sensorTimeMap {
						if v.After(lastActionTime) {
							lastActionTime = v
						}
					}
				}
				sensorLock.RUnlock()

				if time.Since(lastActionTime).Seconds() > timeout.Seconds() {
					fmt.Printf("Exited Sensor Pod %s due to no actions in the last %v\n", podName, o.idleTimeout)
					return
				}
				select {
				case <-ctx.Done():
					return
				case data, ok := <-sCh:
					if !ok {
						return
					}
					// e.g. triggerName/success
					t := strings.Split(data, "/")
					sensorLock.Lock()
					if _, ok := sensorMap[data]; !ok {
						sensorMap[t[0]+"/"+Success] = 0
						sensorMap[t[0]+"/"+Failure] = 0
					}
					if t[1] == Success {
						sensorMap[t[0]+"/"+Success]++
					} else {
						sensorMap[t[0]+"/"+Failure]++
					}
					if sensorMap[t[0]+"/"+Success]+sensorMap[t[0]+"/"+Failure] == 1 {
						sensorTimeMap[t[0]+"/"+First] = time.Now()
					}
					sensorTimeMap[t[0]+"/"+Last] = time.Now()
					sensorLock.Unlock()
				default:
				}
			}
		}(sensorPodName)
	}

	for _, esPodName := range esPodNames {
		wg.Add(1)
		go func(podName string) {
			defer wg.Done()
			fmt.Printf("Started watching EventSource Pod %s ...\n", podName)
			stream, err := o.kubeClient.CoreV1().Pods(o.namespace).GetLogs(podName, &corev1.PodLogOptions{Follow: true}).Stream(ctx)
			if err != nil {
				fmt.Printf("failed to acquire event source pod %s log stream: %v", podName, err)
				return
			}
			defer func() { _ = stream.Close() }()

			sCh := make(chan string)
			go func(dataCh chan string) {
				s := bufio.NewScanner(stream)
				for {
					if !s.Scan() {
						fmt.Printf("Can not read: %v\n", s.Err())
						close(dataCh)
						return
					}
					data := s.Bytes()
					eventName := getLogValue(data, startTime, EventNameKey)
					if eventName == "" {
						continue
					}
					if successEventReg.Match(data) {
						dataCh <- eventName + "/" + Success
					} else if failureEventReg.Match(data) {
						dataCh <- eventName + "/" + Failure
					}
				}
			}(sCh)

			for {
				if o.hardTimeout != nil && time.Since(startTime).Seconds() > o.hardTimeout.Seconds() {
					fmt.Printf("Exited EventSource Pod %s due to the hard timeout %v\n", podName, *o.hardTimeout)
					return
				}
				timeout := 5 * 60 * time.Second
				lastEventTime := startTime

				esLock.RLock()
				if len(esMap) > 0 && len(esTimeMap) > 0 {
					timeout = o.idleTimeout
					for _, v := range esTimeMap {
						if v.After(lastEventTime) {
							lastEventTime = v
						}
					}
				}
				esLock.RUnlock()
				if time.Since(lastEventTime).Seconds() > timeout.Seconds() {
					fmt.Printf("Exited EventSource Pod %s due to no active events in the last %v\n", podName, o.idleTimeout)
					return
				}
				select {
				case <-ctx.Done():
					return
				case data, ok := <-sCh:
					if !ok {
						return
					}
					// e.g. eventName/success
					t := strings.Split(data, "/")
					esLock.Lock()
					if _, ok := esMap[data]; !ok {
						esMap[t[0]+"/"+Success] = 0
						esMap[t[0]+"/"+Failure] = 0
					}
					if t[1] == Success {
						esMap[t[0]+"/"+Success]++
					} else {
						esMap[t[0]+"/"+Failure]++
					}
					if esMap[t[0]+"/"+Success]+esMap[t[0]+"/"+Failure] == 1 {
						esTimeMap[t[0]+"/"+First] = time.Now()
					}
					esTimeMap[t[0]+"/"+Last] = time.Now()
					esLock.Unlock()
				default:
				}
			}
		}(esPodName)
	}

	wg.Wait()

	time.Sleep(3 * time.Second)
	eventNames := []string{}
	for k := range esMap {
		t := strings.Split(k, "/")
		if t[1] == Success {
			eventNames = append(eventNames, t[0])
		}
	}
	triggerNames := []string{}
	for k := range sensorMap {
		t := strings.Split(k, "/")
		if t[1] == Success {
			triggerNames = append(triggerNames, t[0])
		}
	}
	fmt.Printf("\n++++++++++++++++++++++++ Events Summary +++++++++++++++++++++++\n")
	if len(eventNames) == 0 {
		fmt.Println("No events.")
	} else {
		for _, eventName := range eventNames {
			fmt.Printf("Event Name                    : %s\n", eventName)
			fmt.Printf("Total processed events        : %d\n", esMap[eventName+"/"+Success]+esMap[eventName+"/"+Failure])
			fmt.Printf("Events sent successful        : %d\n", esMap[eventName+"/"+Success])
			fmt.Printf("Events sent failed            : %d\n", esMap[eventName+"/"+Failure])
			fmt.Printf("First event sent at           : %v\n", esTimeMap[eventName+"/"+First])
			fmt.Printf("Last event sent at            : %v\n", esTimeMap[eventName+"/"+Last])
			fmt.Printf("Total time taken              : %v\n", esTimeMap[eventName+"/"+Last].Sub(esTimeMap[eventName+"/"+First]))
			fmt.Println("--")
		}
	}

	fmt.Printf("\n+++++++++++++++++++++++ Actions Summary +++++++++++++++++++++++\n")
	if len(triggerNames) == 0 {
		fmt.Println("No actions.")
	} else {
		for _, triggerName := range triggerNames {
			fmt.Printf("Trigger Name                  : %s\n", triggerName)
			fmt.Printf("Total triggered actions       : %d\n", sensorMap[triggerName+"/"+Success]+sensorMap[triggerName+"/"+Failure])
			fmt.Printf("Action triggered successfully : %d\n", sensorMap[triggerName+"/"+Success])
			fmt.Printf("Action triggered failed       : %d\n", sensorMap[triggerName+"/"+Failure])
			fmt.Printf("First action triggered at     : %v\n", sensorTimeMap[triggerName+"/"+First])
			fmt.Printf("Last action triggered at      : %v\n", sensorTimeMap[triggerName+"/"+Last])
			fmt.Printf("Total time taken              : %v\n", sensorTimeMap[triggerName+"/"+Last].Sub(sensorTimeMap[triggerName+"/"+First]))
			fmt.Println("--")
		}
	}
	return nil
}

// Check if it a valid log in JSON format, which contains something
// like `"ts":1616093369.2583323`, and if it's later than start,
// return the value of a log key
func getLogValue(log []byte, start time.Time, key string) string {
	t := make(map[string]interface{})
	if err := json.Unmarshal(log, &t); err != nil {
		// invalid json format log
		return ""
	}
	ts, ok := t["ts"]
	if !ok {
		return ""
	}
	s, ok := ts.(float64)
	if !ok {
		return ""
	}
	if float64(start.Unix()) > s {
		// old log
		return ""
	}
	v, ok := t[key]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func (o *options) dynamicFor(r schema.GroupVersionResource) dynamic.ResourceInterface {
	resourceInterface := dynamic.NewForConfigOrDie(o.restConfig).Resource(r)
	return resourceInterface.Namespace(o.namespace)
}

func (o *options) cleanUpResources(ctx context.Context) error {
	hasTestLabel := metav1.ListOptions{LabelSelector: StressTestingLabel}
	resources := []schema.GroupVersionResource{
		{Group: v1alpha1.SchemeGroupVersion.Group, Version: v1alpha1.SchemeGroupVersion.Version, Resource: "eventsources"},
		{Group: v1alpha1.SchemeGroupVersion.Group, Version: v1alpha1.SchemeGroupVersion.Version, Resource: "sensors"},
		{Group: v1alpha1.SchemeGroupVersion.Group, Version: v1alpha1.SchemeGroupVersion.Version, Resource: "eventbus"},
	}
	for _, r := range resources {
		if err := o.dynamicFor(r).DeleteCollection(ctx, metav1.DeleteOptions{PropagationPolicy: &background}, hasTestLabel); err != nil {
			return err
		}
	}

	for _, r := range resources {
		for {
			list, err := o.dynamicFor(r).List(ctx, hasTestLabel)
			if err != nil {
				return err
			}
			if len(list.Items) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}

func (o *options) Start(ctx context.Context) error {
	fmt.Println("#################### Preparing ####################")

	esName := o.esName
	sensorName := o.sensorName

	// Need to create
	if esName == "" && sensorName == "" {
		// Clean up resources if any
		if err := o.cleanUpResources(ctx); err != nil {
			return err
		}
		time.Sleep(10 * time.Second)

		// Create EventBus
		eb, err := o.createEventBus(ctx)
		if err != nil {
			return err
		}

		defer func() {
			if !o.noCleanUp {
				_ = o.eventBusClient.Delete(ctx, eb.Name, metav1.DeleteOptions{})
			}
		}()

		time.Sleep(5 * time.Second)

		// Create Sensor
		sensor, err := o.createSensor(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if !o.noCleanUp {
				_ = o.sensorClient.Delete(ctx, sensor.Name, metav1.DeleteOptions{})
			}
		}()

		// Create Event Source
		es, err := o.createEventSource(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if !o.noCleanUp {
				_ = o.eventSourceClient.Delete(ctx, es.Name, metav1.DeleteOptions{})
			}
		}()

		esName = es.Name
		sensorName = sensor.Name
	} else {
		fmt.Printf("------- Use existing EventSource and Sensor -------\n")
		fmt.Printf("EventSource name : %s\n", esName)
		fmt.Printf("Sensor name      : %s\n", sensorName)
	}

	// Run testing
	fmt.Println("")
	fmt.Println("################# Started Testing #################")

	return o.runTesting(ctx, esName, sensorName)
}

func readResource(text string, v metav1.Object) error {
	var data []byte
	var err error
	if strings.HasPrefix(text, "@") {
		file := strings.TrimPrefix(text, "@")
		_, fileName, _, _ := runtime.Caller(0)
		data, err = os.ReadFile(filepath.Dir(fileName) + "/" + file)
		if err != nil {
			return fmt.Errorf("failed to read a file: %w", err)
		}
	} else {
		data = []byte(text)
	}
	if err = yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal the yaml: %w", err)
	}
	return nil
}

func getTestingEventSource(str string) TestingEventSource {
	switch str {
	case "webhook":
		return WebhookEventSource
	case "sqs":
		return SQSEventSource
	case "sns":
		return SNSEventSource
	case "kafka":
		return KafkaEventSource
	case "redis":
		return RedisEventSource
	case "nats":
		return NATSEventSource
	default:
		return UnsupportedEventsource
	}
}

func getEventBusType(str string) EventBusType {
	switch str {
	case "jetstream":
		return JetstreamEventBus
	case "stan":
		return STANEventBus
	default:
		return UnsupportedEventBusType
	}
}

func getTestingTrigger(str string) TestingTrigger {
	switch str {
	case "log":
		return LogTrigger
	case "workflow":
		return WorkflowTrigger
	default:
		return UnsupportedTrigger
	}
}

func main() {
	var (
		ebTypeStr      string
		esTypeStr      string
		triggerTypeStr string
		esName         string
		sensorName     string
		idleTimeoutStr string
		hardTimeoutStr string
		noCleanUp      bool
	)
	var rootCmd = &cobra.Command{
		Use:   "go run ./test/stress/main.go",
		Short: "Argo Events stress testing.",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			esType := getTestingEventSource(esTypeStr)
			triggerType := getTestingTrigger(triggerTypeStr)
			if esName == "" && sensorName == "" {
				if esType == UnsupportedEventsource {
					fmt.Printf("Invalid event source %s\n\n", esTypeStr)
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}

				if triggerType == UnsupportedTrigger {
					fmt.Printf("Invalid trigger %s\n\n", triggerTypeStr)
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}
			} else if (esName == "" && sensorName != "") || (esName != "" && sensorName == "") {
				fmt.Printf("Both event source name and sensor name need to be specified\n\n")
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			eventBusType := getEventBusType(ebTypeStr)
			if eventBusType == UnsupportedEventBusType {
				fmt.Printf("Invalid event bus type %s\n\n", ebTypeStr)
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			idleTimeout, err := time.ParseDuration(idleTimeoutStr)
			if err != nil {
				fmt.Printf("Invalid idle timeout %s: %v\n\n", idleTimeoutStr, err)
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			opts, err := NewOptions(esType, triggerType, eventBusType, esName, sensorName, idleTimeout, noCleanUp)
			if err != nil {
				fmt.Printf("Failed: %v\n", err)
				os.Exit(1)
			}
			if hardTimeoutStr != "" {
				hardTimeout, err := time.ParseDuration(hardTimeoutStr)
				if err != nil {
					fmt.Printf("Invalid hard timeout %s: %v\n\n", hardTimeoutStr, err)
					cmd.HelpFunc()(cmd, args)
					os.Exit(1)
				}
				opts.hardTimeout = &hardTimeout
			}
			ctx := signals.SetupSignalHandler()
			if err = opts.Start(ctx); err != nil {
				panic(err)
			}
		},
	}
	rootCmd.Flags().StringVarP(&ebTypeStr, "eb-type", "b", "", "Type of event bus to be tested: stan, jetstream")
	rootCmd.Flags().StringVarP(&esTypeStr, "es-type", "e", "", "Type of event source to be tested, e.g. webhook, sqs, etc.")
	rootCmd.Flags().StringVarP(&triggerTypeStr, "trigger-type", "t", string(LogTrigger), "Type of trigger to be tested, e.g. log, workflow.")
	rootCmd.Flags().StringVar(&esName, "es-name", "", "Name of an existing event source to be tested")
	rootCmd.Flags().StringVar(&sensorName, "sensor-name", "", "Name of an existing sensor to be tested.")
	rootCmd.Flags().StringVar(&idleTimeoutStr, "idle-timeout", "60s", "Exit in a period of time without any active events or actions. e.g. 30s, 2m.")
	rootCmd.Flags().StringVar(&hardTimeoutStr, "hard-timeout", "", "Exit in a period of time after the testing starts. e.g. 120s, 5m. If it's specified, the application will exit at the time either hard-timeout or idle-timeout meets.")
	rootCmd.Flags().BoolVar(&noCleanUp, "no-clean-up", false, "Whether to clean up the created resources.")

	_ = rootCmd.Execute()
}
