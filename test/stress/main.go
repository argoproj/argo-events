package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	"github.com/argoproj/argo-events/pkg/apis/eventbus"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/eventsource"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
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
	eventBusName   = "stress-testing"
	defaultTimeout = 60 * time.Second

	Success = "success"
	Failure = "failure"

	StressTestingLabel      = "argo-events-stress"
	StressTestingLabelValue = "true"

	logEventSourceStarted      = "Eventing server started."
	logSensorStarted           = "Sensor started."
	logTriggerActionSuccessful = "successfully processed the trigger"
	logTriggerActionFailed     = "failed to trigger action"
	logEventSuccessful         = "succeeded to publish an event"
	logEventFailed             = "failed to publish an event"
)

type TestingEventSource string

// possible values of TestingEventSource
const (
	UnsupportedEventsource TestingEventSource = "unsupported"
	WebhookEventSource     TestingEventSource = "webhook"
	SQSEventSource         TestingEventSource = "sqs"
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
	esName             string
	sensorName         string
	// Inactive time before exiting
	idleTimeout time.Duration
	hardTimeout *time.Duration
	noCleanUp   bool

	kubeClient        kubernetes.Interface
	eventBusClient    eventbuspkg.EventBusInterface
	eventSourceClient eventsourcepkg.EventSourceInterface
	sensorClient      sensorpkg.SensorInterface
	restConfig        *rest.Config
}

func NewOptions(testingEventSource TestingEventSource, testingTrigger TestingTrigger, esName, sensorName string, idleTimeout time.Duration, noCleanUp bool) (*options, error) {
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
	eventBusClient := eventbusversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().EventBus(namespace)
	eventSourceClient := eventsourceversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().EventSources(namespace)
	sensorClient := sensorversiond.NewForConfigOrDie(config).ArgoprojV1alpha1().Sensors(namespace)
	return &options{
		namespace:          namespace,
		testingEventSource: testingEventSource,
		testingTrigger:     testingTrigger,
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

func (o *options) createEventBus(ctx context.Context) (*eventbusv1alpha1.EventBus, error) {
	fmt.Println("------- Creating EventBus -------")
	eb := &eventbusv1alpha1.EventBus{}
	if err := readResource("@testdata/eventbus/default.yaml", eb); err != nil {
		return nil, fmt.Errorf("failed to read event bus yaml file: %w", err)
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

func (o *options) createEventSource(ctx context.Context) (*eventsourcev1alpha1.EventSource, error) {
	fmt.Printf("\n------- Creating %v EventSource -------\n", o.testingEventSource)
	es := &eventsourcev1alpha1.EventSource{}
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
	contains, err := testutil.EventSourcePodLogContains(ctx, o.kubeClient, o.namespace, es.Name, logEventSourceStarted, defaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("expected to see event source pod contains something: %w", err)
	}
	if !contains {
		return nil, fmt.Errorf("EventSource Pod does look good, it might have started failed")
	}
	return result, nil
}

func (o *options) createSensor(ctx context.Context) (*sensorv1alpha1.Sensor, error) {
	fmt.Printf("\n------- Creating %v Sensor -------\n", o.testingTrigger)
	sensor := &sensorv1alpha1.Sensor{}
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
	contains, err := testutil.SensorPodLogContains(ctx, o.kubeClient, o.namespace, sensor.Name, logSensorStarted, defaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("expected to see sensor pod contains something: %w", err)
	}
	if !contains {
		return nil, fmt.Errorf("Sensor Pod does look good, it might have started failed")
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

	successActionReg, err := regexp.Compile(logTriggerActionSuccessful)
	if err != nil {
		return fmt.Errorf("failed to compile regex for sensor success pattern: %v", err)
	}
	failureActionReg, err := regexp.Compile(logTriggerActionFailed)
	if err != nil {
		return fmt.Errorf("failed to compile regex for sensor failure pattern: %v", err)
	}

	successEventReg, err := regexp.Compile(logEventSuccessful)
	if err != nil {
		return fmt.Errorf("failed to compile regex for event source success pattern: %v", err)
	}
	failureEventReg, err := regexp.Compile(logEventFailed)
	if err != nil {
		return fmt.Errorf("failed to compile regex for event source failure pattern: %v", err)
	}

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

	esMap := map[string]int64{Success: 0, Failure: 0}
	var firstEventTime time.Time
	lastEventTime := time.Now()

	sensorMap := map[string]int64{Success: 0, Failure: 0}
	var firstActionTime time.Time
	lastActionTime := time.Now()

	var esLock = &sync.Mutex{}
	var sensorLock = &sync.Mutex{}

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

			sCh := make(chan bool)
			go func(successCh chan bool) {
				s := bufio.NewScanner(stream)
				for {
					if !s.Scan() {
						fmt.Printf("Can not read: %v\n", s.Err())
						close(successCh)
						return
					}
					data := s.Bytes()
					if successActionReg.Match(data) && isValid(data, startTime) {
						successCh <- true
					} else if failureActionReg.Match(data) && isValid(data, startTime) {
						successCh <- false
					}
				}
			}(sCh)

			for {
				if o.hardTimeout != nil && time.Since(startTime).Seconds() > o.hardTimeout.Seconds() {
					fmt.Printf("Exited Sensor Pod %s due to the hard timeout %v\n", podName, *o.hardTimeout)
					return
				}
				timeout := o.idleTimeout
				if sensorMap[Success]+sensorMap[Failure] == 0 {
					timeout = 5 * 60 * time.Second
				}
				if time.Since(lastActionTime).Seconds() > timeout.Seconds() {
					fmt.Printf("Exited Sensor Pod %s due to no actions in the last %v\n", podName, o.idleTimeout)
					return
				}
				select {
				case <-ctx.Done():
					return
				case successful, ok := <-sCh:
					if !ok {
						return
					}
					sensorLock.Lock()
					if successful {
						sensorMap[Success]++
					} else {
						sensorMap[Failure]++
					}
					if sensorMap[Success]+sensorMap[Failure] == 1 {
						firstActionTime = time.Now()
					}
					lastActionTime = time.Now()
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

			sCh := make(chan bool)
			go func(successCh chan bool) {
				s := bufio.NewScanner(stream)
				for {
					if !s.Scan() {
						fmt.Printf("Can not read: %v\n", s.Err())
						close(successCh)
						return
					}
					data := s.Bytes()
					if successEventReg.Match(data) && isValid(data, startTime) {
						successCh <- true
					} else if failureEventReg.Match(data) && isValid(data, startTime) {
						successCh <- false
					}
				}
			}(sCh)

			for {
				if o.hardTimeout != nil && time.Since(startTime).Seconds() > o.hardTimeout.Seconds() {
					fmt.Printf("Exited EventSource Pod %s due to the hard timeout %v\n", podName, *o.hardTimeout)
					return
				}
				timeout := o.idleTimeout
				if esMap[Success]+esMap[Failure] == 0 {
					timeout = 5 * 60 * time.Second
				}
				if time.Since(lastEventTime).Seconds() > timeout.Seconds() {
					fmt.Printf("Exited EventSource Pod %s due to no active events in the last %v\n", podName, o.idleTimeout)
					return
				}
				select {
				case <-ctx.Done():
					return
				case successful, ok := <-sCh:
					if !ok {
						return
					}
					esLock.Lock()
					if successful {
						esMap[Success]++
					} else {
						esMap[Failure]++
					}
					if esMap[Success]+esMap[Failure] == 1 {
						firstEventTime = time.Now()
					}
					lastEventTime = time.Now()
					esLock.Unlock()
				default:
				}
			}
		}(esPodName)
	}

	wg.Wait()

	time.Sleep(3 * time.Second)

	fmt.Printf("\n++++++++++++++++++++++++ Events Summary +++++++++++++++++++++++\n")
	fmt.Printf("Total processed events        : %d\n", esMap[Success]+esMap[Failure])
	fmt.Printf("Events sent successful        : %d\n", esMap[Success])
	fmt.Printf("Events sent failed            : %d\n", esMap[Failure])
	fmt.Printf("First event sent at           : %v\n", firstEventTime)
	fmt.Printf("Last event sent at            : %v\n", lastEventTime)
	fmt.Printf("Total time taken              : %v\n", lastEventTime.Sub(firstEventTime))

	fmt.Printf("\n+++++++++++++++++++++++ Actions Summary +++++++++++++++++++++++\n")
	fmt.Printf("Total triggered actions       : %d\n", sensorMap[Success]+sensorMap[Failure])
	fmt.Printf("Action triggered successfully : %d\n", sensorMap[Success])
	fmt.Printf("Action triggered failed       : %d\n", sensorMap[Failure])
	fmt.Printf("First action triggered at     : %v\n", firstActionTime)
	fmt.Printf("Last action triggered at      : %v\n", lastActionTime)
	fmt.Printf("Total time taken              : %v\n", lastActionTime.Sub(firstActionTime))
	return nil
}

// Check if it a valid log in JSON format, which contains something
// like `"ts":1616093369.2583323`, and return if it's later than start.
func isValid(log []byte, start time.Time) bool {
	t := make(map[string]interface{})
	if err := json.Unmarshal(log, &t); err != nil {
		fmt.Println(err)
		return false
	}
	ts, ok := t["ts"]
	if !ok {
		return false
	}
	s, ok := ts.(float64)
	if !ok {
		return false
	}
	return float64(start.Unix()) < s
}

func (o *options) dynamicFor(r schema.GroupVersionResource) dynamic.ResourceInterface {
	resourceInterface := dynamic.NewForConfigOrDie(o.restConfig).Resource(r)
	return resourceInterface.Namespace(o.namespace)
}

func (o *options) cleanUpResources(ctx context.Context) error {
	hasTestLabel := metav1.ListOptions{LabelSelector: StressTestingLabel}
	resources := []schema.GroupVersionResource{
		{Group: eventsource.Group, Version: "v1alpha1", Resource: eventsource.Plural},
		{Group: sensor.Group, Version: "v1alpha1", Resource: sensor.Plural},
		{Group: eventbus.Group, Version: "v1alpha1", Resource: eventbus.Plural},
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
	fmt.Println("")

	return o.runTesting(ctx, esName, sensorName)
}

func readResource(text string, v metav1.Object) error {
	var data []byte
	var err error
	if strings.HasPrefix(text, "@") {
		file := strings.TrimPrefix(text, "@")
		_, fileName, _, _ := runtime.Caller(0)
		data, err = ioutil.ReadFile(filepath.Dir(fileName) + "/" + file)
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
	default:
		return UnsupportedEventsource
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

			idleTimeout, err := time.ParseDuration(idleTimeoutStr)
			if err != nil {
				fmt.Printf("Invalid idle timeout %s: %v\n\n", idleTimeoutStr, err)
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			opts, err := NewOptions(esType, triggerType, esName, sensorName, idleTimeout, noCleanUp)
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
	rootCmd.Flags().StringVarP(&esTypeStr, "es-type", "e", "", "Type of event source to be tested, e.g. webhook, sqs, etc.")
	rootCmd.Flags().StringVarP(&triggerTypeStr, "trigger-type", "t", string(LogTrigger), "Type of trigger to be tested, e.g. log, workflow.")
	rootCmd.Flags().StringVar(&esName, "es-name", "", "Name of an existing event source to be tested")
	rootCmd.Flags().StringVar(&sensorName, "sensor-name", "", "Name of an existing sensor to be tested.")
	rootCmd.Flags().StringVar(&idleTimeoutStr, "idle-timeout", "60s", "Exit in how long without any active events or actions. e.g. 30s, 2m.")
	rootCmd.Flags().StringVar(&hardTimeoutStr, "hard-timeout", "", "Exit in how long after the stress testing starts. e.g. 120s, 5m. If it's specified, the application will exit at the time either hard-timeout or idle-timeout meets.")
	rootCmd.Flags().BoolVar(&noCleanUp, "no-clean-up", false, "Whether to clean up the created resources.")

	_ = rootCmd.Execute()
}
