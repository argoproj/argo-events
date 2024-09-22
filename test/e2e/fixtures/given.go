package fixtures

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventspkg "github.com/argoproj/argo-events/pkg/client/clientset/versioned/typed/events/v1alpha1"
)

type Given struct {
	t                 *testing.T
	eventBusClient    eventspkg.EventBusInterface
	eventSourceClient eventspkg.EventSourceInterface
	sensorClient      eventspkg.SensorInterface
	eventBus          *v1alpha1.EventBus
	eventSource       *v1alpha1.EventSource
	sensor            *v1alpha1.Sensor
	restConfig        *rest.Config
	kubeClient        kubernetes.Interface
}

// creates an EventBus based on the parameter, this may be:
//
// 1. A file name if it starts with "@"
// 2. Raw YAML.
func (g *Given) EventBus(text string) *Given {
	g.t.Helper()
	g.eventBus = &v1alpha1.EventBus{}
	g.readResource(text, g.eventBus)
	l := g.eventBus.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[Label] = LabelValue
	g.eventBus.SetLabels(l)
	g.eventBus.SetName(EventBusName)
	return g
}

// creates an EventSource based on the parameter, this may be:
//
// 1. A file name if it starts with "@"
// 2. Raw YAML.
func (g *Given) EventSource(text string) *Given {
	g.t.Helper()
	g.eventSource = &v1alpha1.EventSource{}
	g.readResource(text, g.eventSource)
	l := g.eventSource.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[Label] = LabelValue
	g.eventSource.SetLabels(l)
	g.eventSource.Spec.EventBusName = EventBusName
	return g
}

// creates a Sensor based on the parameter, this may be:
//
// 1. A file name if it starts with "@"
// 2. Raw YAML.
func (g *Given) Sensor(text string) *Given {
	g.t.Helper()
	g.sensor = &v1alpha1.Sensor{}
	g.readResource(text, g.sensor)
	l := g.sensor.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[Label] = LabelValue
	g.sensor.SetLabels(l)
	g.sensor.Spec.EventBusName = EventBusName
	return g
}

func (g *Given) readResource(text string, v metav1.Object) {
	g.t.Helper()
	var file string
	if strings.HasPrefix(text, "@") {
		file = strings.TrimPrefix(text, "@")
	} else {
		f, err := os.CreateTemp("", "argo-events-e2e")
		if err != nil {
			g.t.Fatal(err)
		}
		_, err = f.Write([]byte(text))
		if err != nil {
			g.t.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			g.t.Fatal(err)
		}
		file = f.Name()
	}

	f, err := os.ReadFile(file)
	if err != nil {
		g.t.Fatal(err)
	}
	err = yaml.Unmarshal(f, v)
	if err != nil {
		g.t.Fatal(err)
	}
}

func (g *Given) When() *When {
	return &When{
		t:                 g.t,
		eventBusClient:    g.eventBusClient,
		eventSourceClient: g.eventSourceClient,
		sensorClient:      g.sensorClient,
		eventBus:          g.eventBus,
		eventSource:       g.eventSource,
		sensor:            g.sensor,
		restConfig:        g.restConfig,
		kubeClient:        g.kubeClient,
	}
}

var OutputRegexp = func(rx string) func(t *testing.T, output string, err error) {
	return func(t *testing.T, output string, err error) {
		t.Helper()
		if assert.NoError(t, err, output) {
			assert.Regexp(t, rx, output)
		}
	}
}
