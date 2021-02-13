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
	"github.com/argoproj/argo-events/pkg/apis/eventbus"
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
	defaultTimeout = 30 * time.Second
)

var (
	foreground = metav1.DeletePropagationForeground
	background = metav1.DeletePropagationBackground
)

type E2ESuite struct {
	suite.Suite
	RestConfig        *rest.Config
	eventBusClient    eventbuspkg.EventBusInterface
	eventSourceClient eventsourcepkg.EventSourceInterface
	sensorClient      sensorpkg.SensorInterface
	KubeClient        kubernetes.Interface
}

func (s *E2ESuite) SetupSuite() {
	var err error
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	s.RestConfig, err = common.GetClientConfig(kubeConfig)
	s.CheckError(err)
	s.KubeClient, err = kubernetes.NewForConfig(s.RestConfig)
	s.CheckError(err)
	s.eventBusClient = eventbusversiond.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().EventBus(Namespace)
	s.eventSourceClient = eventsourceversiond.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().EventSources(Namespace)
	s.sensorClient = sensorversiond.NewForConfigOrDie(s.RestConfig).ArgoprojV1alpha1().Sensors(Namespace)
}

func (s *E2ESuite) TearDownSuite() {
}

func (s *E2ESuite) BeforeTest(string, string) {
	s.DeleteResources()
}

func (s *E2ESuite) DeleteResources() {
	hasTestLabel := metav1.ListOptions{LabelSelector: Label}
	resources := []schema.GroupVersionResource{
		{Group: eventbus.Group, Version: "v1alpha1", Resource: eventbus.Plural},
		{Group: eventsource.Group, Version: "v1alpha1", Resource: eventsource.Plural},
		{Group: sensor.Group, Version: "v1alpha1", Resource: sensor.Plural},
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
	resourceInterface := dynamic.NewForConfigOrDie(s.RestConfig).Resource(r)
	return resourceInterface.Namespace(Namespace)
}

func (s *E2ESuite) CheckError(err error) {
	s.T().Helper()
	if err != nil {
		s.T().Fatal(err)
	}
}
