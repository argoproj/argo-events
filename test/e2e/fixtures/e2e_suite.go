package fixtures

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	versiond "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	eventspkg "github.com/argoproj/argo-events/pkg/client/clientset/versioned/typed/events/v1alpha1"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

const (
	Namespace      = "argo-events"
	Label          = "argo-events-e2e"
	LabelValue     = "true"
	EventBusName   = "argo-events-e2e"
	defaultTimeout = 60 * time.Second
)

var (
	background = metav1.DeletePropagationBackground

	E2EEventBusSTAN = `apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native:
      auth: token`

	E2EEventBusJetstream = `apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  jetstream:
    version: latest`

	E2EEventBusKafka = `apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  kafka:
    url: kafka-broker:9092`
)

type E2ESuite struct {
	suite.Suite
	restConfig        *rest.Config
	eventBusClient    eventspkg.EventBusInterface
	eventSourceClient eventspkg.EventSourceInterface
	sensorClient      eventspkg.SensorInterface
	kubeClient        kubernetes.Interface
}

func (s *E2ESuite) SetupSuite() {
	var err error

	kubeConfig, found := os.LookupEnv(v1alpha1.EnvVarKubeConfig)
	if !found {
		home, _ := os.UserHomeDir()
		kubeConfig = home + "/.kube/config"
		if _, err := os.Stat(kubeConfig); err != nil && os.IsNotExist(err) {
			kubeConfig = ""
		}
	}
	s.restConfig, err = sharedutil.GetClientConfig(kubeConfig)
	s.CheckError(err)
	s.kubeClient, err = kubernetes.NewForConfig(s.restConfig)
	s.CheckError(err)
	s.eventBusClient = versiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().EventBus(Namespace)
	s.eventSourceClient = versiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().EventSources(Namespace)
	s.sensorClient = versiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().Sensors(Namespace)

	// Clean up resources if any
	s.DeleteResources()
	// Clean up test event bus if any
	resources := []schema.GroupVersionResource{
		{Group: v1alpha1.SchemeGroupVersion.Group, Version: v1alpha1.SchemeGroupVersion.Version, Resource: "eventbus"},
	}
	s.deleteResources(resources)

	s.Given().EventBus(GetBusDriverSpec()).
		When().
		CreateEventBus().
		WaitForEventBusReady()
	s.T().Log("EventBus is ready")

	time.Sleep(10 * time.Second) // give it a little extra time to be fully ready // todo: any issues with this? Otherwise, I need to increase the allowance in the backoff
}

func (s *E2ESuite) TearDownSuite() {
	s.DeleteResources()
	s.Given().EventBus(GetBusDriverSpec()).
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

func (s *E2ESuite) deleteResources(resources []schema.GroupVersionResource) {
	hasTestLabel := metav1.ListOptions{LabelSelector: Label}
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

func (s *E2ESuite) DeleteResources() {
	resources := []schema.GroupVersionResource{
		{Group: v1alpha1.SchemeGroupVersion.Group, Version: v1alpha1.SchemeGroupVersion.Version, Resource: "eventsources"},
		{Group: v1alpha1.SchemeGroupVersion.Group, Version: v1alpha1.SchemeGroupVersion.Version, Resource: "sensors"},
		{Group: "", Version: "v1", Resource: "pods"},
	}
	s.deleteResources(resources)
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
		restConfig:        s.restConfig,
		kubeClient:        s.kubeClient,
	}
}

func GetBusDriverSpec() string {
	x := strings.ToUpper(os.Getenv("EventBusDriver"))
	switch x {
	case "JETSTREAM":
		return E2EEventBusJetstream
	case "KAFKA":
		return E2EEventBusKafka
	default:
		return E2EEventBusSTAN
	}
}
