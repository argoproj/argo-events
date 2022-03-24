package fixtures

import (
	"context"
	"fmt"
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
	Namespace = "argo-events"
	//Label     = "argo-events-stan-e2e"
	//Label      = "argo-events-e2e"
	LabelValue       = "true"
	STANEventBusName = "argo-events-stan-e2e"
	//STANEventBusName      = "argo-events-e2e"
	JetstreamEventBusName = "argo-events-js-e2e"
	defaultTimeout        = 60 * time.Second
)

var (
	background = metav1.DeletePropagationBackground

	e2eEventBusSTAN = `apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native:
      auth: token`

	e2eEventBusJS = `apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: jetstream
spec:
  jetstream:
    version: 2.7.3
    replicas: 3`
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

	kubeConfig, found := os.LookupEnv(common.EnvVarKubeConfig)
	if !found {
		home, _ := os.UserHomeDir()
		kubeConfig = home + "/.kube/config"
		if _, err := os.Stat(kubeConfig); err != nil && os.IsNotExist(err) {
			kubeConfig = ""
		}
	}
	s.restConfig, err = common.GetClientConfig(kubeConfig)
	s.CheckError(err)
	s.kubeClient, err = kubernetes.NewForConfig(s.restConfig)
	s.CheckError(err)
	s.eventBusClient = eventbusversiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().EventBus(Namespace)
	s.eventSourceClient = eventsourceversiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().EventSources(Namespace)
	s.sensorClient = sensorversiond.NewForConfigOrDie(s.restConfig).ArgoprojV1alpha1().Sensors(Namespace)

	// Clean up resources if any
	s.DeleteResources()
	// Clean up test event bus if any
	resources := []schema.GroupVersionResource{
		{Group: eventsource.Group, Version: "v1alpha1", Resource: eventbus.Plural},
	}
	s.deleteResources(resources)

	s.Given().EventBus(e2eEventBusSTAN, STANEventBusName).
		When().
		CreateEventBus().
		WaitForEventBusReady()
	s.T().Log("EventBus (STAN) is ready")

	s.Given().EventBus(e2eEventBusJS, JetstreamEventBusName).
		When().
		CreateEventBus().
		WaitForEventBusReady()
	s.T().Log("EventBus (Jetstream) is ready")

	time.Sleep(10 * time.Second)
}

func (s *E2ESuite) TearDownSuite() {
	s.DeleteResources()
	s.Given().EventBus(e2eEventBusSTAN, STANEventBusName).
		When().
		DeleteEventBus().
		Wait(3 * time.Second).
		Then().
		ExpectEventBusDeleted()
	s.T().Log("STAN EventBus is deleted")
	s.Given().EventBus(e2eEventBusJS, JetstreamEventBusName).
		When().
		DeleteEventBus().
		Wait(3 * time.Second).
		Then().
		ExpectEventBusDeleted()
	s.T().Log("Jetstream EventBus is deleted")
}

func (s *E2ESuite) BeforeTest(string, string) {
	s.DeleteResources()
}

func (s *E2ESuite) deleteResourcesOfLabel(resources []schema.GroupVersionResource, label string) {
	hasTestLabel := metav1.ListOptions{LabelSelector: label}
	ctx := context.Background()
	for _, r := range resources {
		err := s.dynamicFor(r).DeleteCollection(ctx, metav1.DeleteOptions{PropagationPolicy: &background}, hasTestLabel)
		s.CheckError(err)
	}

	for _, r := range resources {
		for {
			fmt.Printf("checking for resource %+v with label %s\n", r, label)
			list, err := s.dynamicFor(r).List(ctx, hasTestLabel)
			s.CheckError(err)
			if len(list.Items) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

}

func (s *E2ESuite) deleteResources(resources []schema.GroupVersionResource) {
	s.deleteResourcesOfLabel(resources, STANEventBusName)
	s.deleteResourcesOfLabel(resources, JetstreamEventBusName)
}

/*
func (s *E2ESuite) deleteResources(resources []schema.GroupVersionResource) {
	hasTestLabel := metav1.ListOptions{LabelSelector: STANEventBusName} //Label}
	ctx := context.Background()
	for _, r := range resources {
		err := s.dynamicFor(r).DeleteCollection(ctx, metav1.DeleteOptions{PropagationPolicy: &background}, hasTestLabel)
		s.CheckError(err)
	}

	for _, r := range resources {
		for {
			fmt.Printf("checking for resource %+v with label %s\n", r, STANEventBusName)
			list, err := s.dynamicFor(r).List(ctx, hasTestLabel)
			s.CheckError(err)
			if len(list.Items) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	hasTestLabel = metav1.ListOptions{LabelSelector: JetstreamEventBusName} //Label}
	for _, r := range resources {
		err := s.dynamicFor(r).DeleteCollection(ctx, metav1.DeleteOptions{PropagationPolicy: &background}, hasTestLabel)
		s.CheckError(err)
	}

	for _, r := range resources {
		for {
			fmt.Printf("checking for resource %+v with label %s\n", r, JetstreamEventBusName)
			list, err := s.dynamicFor(r).List(ctx, hasTestLabel)
			s.CheckError(err)
			if len(list.Items) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}*/

func (s *E2ESuite) DeleteResources() {
	resources := []schema.GroupVersionResource{
		{Group: eventsource.Group, Version: "v1alpha1", Resource: eventsource.Plural},
		{Group: sensor.Group, Version: "v1alpha1", Resource: sensor.Plural},
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
