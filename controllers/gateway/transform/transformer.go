package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	sv1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	zlog "github.com/rs/zerolog"
	suuid "github.com/satori/go.uuid"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"time"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
)

type tConfig struct {
	// EventType is type of the event
	EventType string
	// EventTypeVersion is the version of the `eventType`
	EventTypeVersion string
	// Source of the event
	EventSource string
	// Sensors to dispatch the event to
	Sensors []v1alpha1.SensorNotificationWatcher
	// Gateways to dispatch the event to
	Gateways []v1alpha1.GatewayNotificationWatcher
}

type tOperationCtx struct {
	// Namespace is namespace where gateway-controller is deployed
	Namespace string
	// Event logger
	log zlog.Logger
	// Event configuration
	Config *tConfig
	// Kubernetes clientset
	kubeClientset kubernetes.Interface
}

type TransformerPayload struct {
	// Src contains information about which specific configuration in gateway generated the event
	Src string `json:"src"`
	// Payload is event data
	Payload []byte `json:"payload"`
}

func NewTransformerConfig(eventType string, eventTypeVersion string, eventSource string, sensors []v1alpha1.SensorNotificationWatcher, gateways []v1alpha1.GatewayNotificationWatcher) *tConfig {
	return &tConfig{
		EventType:        eventType,
		EventTypeVersion: eventTypeVersion,
		EventSource:      eventSource,
		Sensors:          sensors,
		Gateways:         gateways,
	}
}

func NewTransformOperationContext(config *tConfig, namespace string, clientset kubernetes.Interface) *tOperationCtx {
	return &tOperationCtx{
		Namespace:     namespace,
		Config:        config,
		kubeClientset: clientset,
		log:           zlog.New(os.Stdout).With().Str("gateway-controller-name", config.EventSource).Logger(),
	}
}

// Transform request transforms http request payload into CloudEvent
func (toc *tOperationCtx) transform(r *http.Request) (*sv1alpha.Event, error) {
	// Generate an event id
	eventId := suuid.Must(suuid.NewV4(), nil)
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Errorf("failed to parse request payload. Err %+v", err)
		return nil, err
	}

	var tp TransformerPayload
	err = json.Unmarshal(payload, &tp)

	if err != nil {
		fmt.Errorf("failed to convert request payload into transformer payload. Err %+v", err)
		return nil, err
	}

	toc.log.Debug().Str("source", tp.Src).Msg("received an event")

	// Create an CloudEvent
	ce := &sv1alpha.Event{
		Context: sv1alpha.EventContext{
			CloudEventsVersion: common.CloudEventsVersion,
			EventID:            fmt.Sprintf("%x", eventId),
			ContentType:        r.Header.Get(common.HeaderContentType),
			EventTime:          metav1.Time{Time: time.Now().UTC()},
			EventType:          toc.Config.EventType,
			EventTypeVersion:   toc.Config.EventTypeVersion,
			Source: &sv1alpha.URI{
				Host: toc.Config.EventSource + "/" + tp.Src,
			},
		},
		Payload: tp.Payload,
	}

	toc.log.Debug().Interface("cloud-event", ce).Msg("transformed event")
	return ce, nil
}

// getWatcherIP returns IP of service which backs given component.
func (toc *tOperationCtx) getWatcherIP(name string) (string, string, error) {
	service, err := toc.kubeClientset.CoreV1().Services(toc.Namespace).Get(common.DefaultSensorServiceName(name), metav1.GetOptions{})
	if err != nil {
		toc.log.Error().Str("service-name", name).Err(err).Msg("failed to connect to watcher service")
		return "", "", err
	}

	switch service.Spec.Type {
	case corev1.ServiceTypeClusterIP:
		return service.ObjectMeta.Name, service.Spec.ClusterIP, nil
	case corev1.ServiceTypeLoadBalancer:
		return service.ObjectMeta.Name, service.Spec.LoadBalancerIP, nil
	case corev1.ServiceTypeNodePort:
		return service.ObjectMeta.Name, service.Spec.ExternalIPs[0], nil
	default:
		// should never come here
		return service.ObjectMeta.Name, "", fmt.Errorf("unknown service type.")
	}
}

// postCloudEventToWatcher makes a HTTP POST call to watcher's service ip
func (toc *tOperationCtx) postCloudEventToWatcher(ip string, port string, endpoint string, payload []byte) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%s%s", ip, port, endpoint), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}

// dispatches the event to interested sensors and gateways
func (toc *tOperationCtx) dispatchTransformedEvent(ce *sv1alpha.Event) error {
	// get the bytes from cloudevent to dispatch to watcher service
	eventBytes, err := json.Marshal(ce)
	if err != nil {
		toc.log.Error().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).Err(err).Msg("failed to marshal event")
		return err
	}

	for _, sensor := range toc.Config.Sensors {
		serviceName, ip, err := toc.getWatcherIP(sensor.Name)
		if err != nil {
			toc.log.Error().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).Str("sensor-name", sensor.Name).Str("sensor-port", common.SensorServicePort).Str("sensor-endpoint", common.SensorServiceEndpoint).Err(err).
				Msg("failed to get watcher ip")
			continue
		}
		// dispatch the event
		toc.log.Info().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
			Str("service-name", serviceName).Str("sensor-name", sensor.Name).Str("sensor-port", common.SensorServicePort).Str("sensor-endpoint", common.SensorServiceEndpoint).Msg("dispatching cloudevent to sensor")
		err = toc.postCloudEventToWatcher(ip, common.SensorServicePort, common.SensorServiceEndpoint,  eventBytes)
		if err != nil {
			toc.log.Error().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).Str("sensor-name", sensor.Name).Err(err).Msg("failed to send event to watcher")
		} else {
			toc.log.Error().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).Str("sensor-name", sensor.Name).Msg("event dispatched to watcher")
		}
	}
	for _, gateway := range toc.Config.Gateways {
		serviceName, ip, err := toc.getWatcherIP(gateway.Name)
		if err != nil {
			toc.log.Error().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).Str("gateway-name", gateway.Name).Str("gateway-port", gateway.Port).Str("gateway-endpoint", gateway.Endpoint).Err(err).
				Msg("failed to get watcher ip")
			continue
		}
		// dispatch the event
		toc.log.Info().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
			Str("service-name", serviceName).Str("gateway-name", gateway.Name).Str("gateway-port", gateway.Port).Str("gateway-endpoint", gateway.Endpoint).Msg("dispatching cloudevent to gateway")
		err = toc.postCloudEventToWatcher(ip, common.SensorServicePort, common.SensorServiceEndpoint,  eventBytes)
		if err != nil {
			toc.log.Error().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).Str("sensor-name", gateway.Name).Err(err).Msg("failed to send event to watcher")
		} else {
			toc.log.Error().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).Str("sensor-name", gateway.Name).Msg("event dispatched to watcher")
		}
	}
	toc.log.Info().Str("event-source", ce.Context.Source.Host).Str("event-id", ce.Context.EventID).
		Str("event-source", ce.Context.Source.Host).Str("", ce.Context.EventID).Msg("event sent to all watchers")
	return nil
}

// transforms the event into cloudevent
func (toc *tOperationCtx) TransformRequest(w http.ResponseWriter, r *http.Request) {
	toc.log.Info().Msg("transforming incoming request into cloudevent")

	// transform event
	ce, err := toc.transform(r)
	if err != nil {
		toc.log.Error().Err(err).Msg("failed to transform user event into CloudEvent")
		common.SendErrorResponse(w)
		return
	}

	// dispatch the cloudevent to sensors registered for this gateway
	err = toc.dispatchTransformedEvent(ce)
	if err != nil {
		toc.log.Error().Err(err).Str("event-id", ce.Context.EventID).Msg("failed to send the event to sensor")
		common.SendErrorResponse(w)
		return
	}
	common.SendSuccessResponse(w)
}
