package fixtures

import (
	"context"
	"os"
	"time"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	eventbusversiond "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned"
	eventbuspkg "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned/typed/eventbus/v1alpha1"
	eventsourceversiond "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/typed/eventsource/v1alpha1"
	sensorversiond "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	sensorpkg "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/typed/sensor/v1alpha1"
)

const (
	Namespace      = "argo-events"
	Label          = "argo-events-e2e"
	EventBusName   = "argo-events-e2e"
	defaultTimeout = 60 * time.Second
)

var (
	foreground = metav1.DeletePropagationForeground
	background = metav1.DeletePropagationBackground

	e2eEventBus = `apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native:
      auth: token`
)

type E2ESuite struct {
	suite.Suite
	restConfig        *rest.Config
	eventBusClient    eventbuspkg.EventBusInterface
	eventSourceClient eventsourcepkg.EventSourceInterface
	sensorClient      sensorpkg.SensorInterface
	kubeClient        kubernetes.Interface
}

func (s *E2ESuite) SetupSuite() {
	var err error
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	s.restConfig, err = common.GetClientConfig(kubeConfig)
	s.CheckError(err)
	s.kubeClient, err = kubernetes.NewForConfig(s.restConfig)
	s.CheckError(err)
	s.eventBusClient = eventbusversiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().EventBus(Namespace)
	s.eventSourceClient = eventsourceversiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().EventSources(Namespace)
	s.sensorClient = sensorversiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().Sensors(Namespace)

	s.Given().EventBus(e2eEventBus).
		When().
		CreateEventBus().
		WaitForEventBusReady().
		WaitForEventBusStatefulSetReady()
	s.T().Log("EventBus is ready")
}

func (s *E2ESuite) TearDownSuite() {
	s.DeleteResources()
	s.Given().EventBus(e2eEventBus).
		When().
		DeleteEventBus().
		Wait(3 * time.Second).
		Then().
		ExpectEventBusDeleted()
	s.T().Log("EventBus is deleted")
}

func (s *E2ESuite) BeforeTest(string, string) {
	s.DeleteResources()
}

func (s *E2ESuite) DeleteResources() {
	hasTestLabel := metav1.ListOptions{LabelSelector: Label}
	resources := []schema.GroupVersionResource{
		{Group: eventsource.Group, Version: "v1alpha1", Resource: eventsource.Plural},
		{Group: sensor.Group, Version: "v1alpha1", Resource: sensor.Plural},
		{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"},
	}

	ctx := context.Background()
	for _, r := range resources {
		err := s.dynamicFor(r).DeleteCollection(ctx, metav1.DeleteOptions{PropagationPolicy: &background}, hasTestLabel)
		s.CheckError(err)
	}

	for _, r := range resources {
		for {
			list, err := s.dynamicFor(r).List(ctx, hasTestLabel)
			s.CheckError(err)
			if len(list.Items) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *E2ESuite) dynamicFor(r schema.GroupVersionResource) dynamic.ResourceInterface {
	resourceInterface := dynamic.NewForConfigOrDie(s.restConfig).Resource(r)
	return resourceInterface.Namespace(Namespace)
}

func (s *E2ESuite) CheckError(err error) {
	s.T().Helper()
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *E2ESuite) Given() *Given {
	return &Given{
		t:                 s.T(),
		eventBusClient:    s.eventBusClient,
		eventSourceClient: s.eventSourceClient,
		sensorClient:      s.sensorClient,
		kubeClient:        s.kubeClient,
	}
}
