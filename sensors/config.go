package sensors

import (
	"github.com/nats-io/go-nats"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	ss_v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	snats "github.com/nats-io/go-nats-streaming"
	"github.com/rs/zerolog"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// sensorExecutionCtx contains execution context for sensor
type sensorExecutionCtx struct {
	// sensorClient is the client for sensor
	sensorClient clientset.Interface
	// kubeClient is the kubernetes client
	kubeClient kubernetes.Interface
	// ClientPool manages a pool of dynamic clients.
	clientPool dynamic.ClientPool
	// DiscoveryClient implements the functions that discover server-supported API groups, versions and resources.
	discoveryClient discovery.DiscoveryInterface
	// sensor object
	sensor *v1alpha1.Sensor
	// http server which exposes the sensor to gateway/s
	server *http.Server
	// logger for the sensor
	log zerolog.Logger
	// queue is internal queue to manage incoming events
	queue chan *updateNotification
	// controllerInstanceID is the instance ID of sensor controller processing this sensor
	controllerInstanceID string
	// updated indicates update to sensor resource
	updated bool
	// nconn is the nats connection
	nconn natsconn
}

type natsconn struct {
	// standard connection
	standard *nats.Conn
	// streaming connection
	stream snats.Conn
}

// updateNotification is servers as a notification message that can be used to update event dependency's state or the sensor resource
type updateNotification struct {
	event            *ss_v1alpha1.Event
	eventDependency  *v1alpha1.EventDependency
	writer           http.ResponseWriter
	sensor           *v1alpha1.Sensor
	notificationType v1alpha1.NotificationType
}

// NewSensorExecutionCtx returns a new sensor execution context.
func NewSensorExecutionCtx(sensorClient clientset.Interface, kubeClient kubernetes.Interface,
	clientPool dynamic.ClientPool, discoveryClient discovery.DiscoveryInterface,
	sensor *v1alpha1.Sensor, controllerInstanceID string) *sensorExecutionCtx {
	return &sensorExecutionCtx{
		sensorClient:         sensorClient,
		kubeClient:           kubeClient,
		clientPool:           clientPool,
		discoveryClient:      discoveryClient,
		sensor:               sensor,
		log:                  common.GetLoggerContext(common.LoggerConf()).Str("sensor-name", sensor.Name).Logger(),
		queue:                make(chan *updateNotification),
		controllerInstanceID: controllerInstanceID,
	}
}
