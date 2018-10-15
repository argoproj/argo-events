package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	sv1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	zlog "github.com/rs/zerolog"
	suuid "github.com/satori/go.uuid"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"time"
)

// tConfig contains information to generate cloudevents specification compliant event
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

// tOperationCtx is the operation context for gateway transformer
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

// TransformerPayload contains payload of cloudevents.
type TransformerPayload struct {
	// Src contains information about which specific configuration in gateway generated the event
	Src string `json:"src"`
	// Payload is event data
	Payload []byte `json:"payload"`
}

// NewTransformerConfig returns a new gateway transformer configuration
func NewTransformerConfig(eventType string, eventTypeVersion string, eventSource string, sensors []v1alpha1.SensorNotificationWatcher, gateways []v1alpha1.GatewayNotificationWatcher) *tConfig {
	return &tConfig{
		EventType:        eventType,
		EventTypeVersion: eventTypeVersion,
		EventSource:      eventSource,
		Sensors:          sensors,
		Gateways:         gateways,
	}
}

// NewTransformOperationContext returns a new gateway transformer operation context
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

	toc.log.Info().Str("source", tp.Src).
		Msg("received an event, converting into cloudevents specification compliant event")

	// Create an CloudEvent
	// See https://github.com/cloudevents/spec for more info.
	ce := &sv1alpha.Event{
		Context: sv1alpha.EventContext{
			CloudEventsVersion: common.CloudEventsVersion,
			EventID:            fmt.Sprintf("%x", eventId),
			ContentType:        "application/json",
			EventTime:          metav1.MicroTime{Time: time.Now().UTC()},
			EventType:          toc.Config.EventType,
			EventTypeVersion:   toc.Config.EventTypeVersion,
			Source: &sv1alpha.URI{
				Host: common.DefaultGatewayConfigurationName(toc.Config.EventSource, tp.Src),
			},
		},
		Payload: tp.Payload,
	}

	toc.log.Info().Interface("event", ce).Msg("transformed into cloud event")
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
		toc.log.Error().
			Str("event-source", ce.Context.Source.Host).
			Str("event-id", ce.Context.EventID).
			Err(err).Msg("failed to marshal event")
		return err
	}

	// dispatch event to sensor watchers
	for _, sensor := range toc.Config.Sensors {
		// get the ip of service backing the sensor
		serviceName, ip, err := toc.getWatcherIP(sensor.Name)
		if err != nil {
			toc.log.Error().
				Str("event-source", ce.Context.Source.Host).
				Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).
				Str("sensor-name", sensor.Name).
				Str("sensor-port", common.SensorServicePort).
				Str("sensor-endpoint", common.SensorServiceEndpoint).
				Err(err).Msg("failed to get watcher ip")
			// maybe service/sensor is not running. skip dispatching event to this sensor
			continue
		}
		// dispatch the event
		toc.log.Info().
			Str("event-source", ce.Context.Source.Host).
			Str("event-id", ce.Context.EventID).
			Str("service-name", serviceName).
			Str("sensor-name", sensor.Name).
			Str("sensor-port", common.SensorServicePort).
			Str("sensor-endpoint", common.SensorServiceEndpoint).Msg("dispatching cloudevent to sensor")

		// send a http post request containing event in the body
		err = toc.postCloudEventToWatcher(ip, common.SensorServicePort, common.SensorServiceEndpoint, eventBytes)
		if err != nil {
			toc.log.Error().
				Str("event-source", ce.Context.Source.Host).
				Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).
				Str("sensor-name", sensor.Name).
				Err(err).Msg("failed to send event to watcher")
		} else {
			toc.log.Error().
				Str("event-source", ce.Context.Source.Host).
				Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).
				Str("sensor-name", sensor.Name).Msg("event dispatched to watcher")
		}
	}

	// dispatch the event to all gateway watchers
	for _, gateway := range toc.Config.Gateways {
		serviceName, ip, err := toc.getWatcherIP(gateway.Name)
		if err != nil {
			toc.log.Error().
				Str("event-source", ce.Context.Source.Host).
				Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).
				Str("gateway-name", gateway.Name).
				Str("gateway-port", gateway.Port).
				Str("gateway-endpoint", gateway.Endpoint).
				Err(err).Msg("failed to get watcher ip")
			// maybe service/gateway is not running
			continue
		}

		toc.log.Info().
			Str("event-source", ce.Context.Source.Host).
			Str("event-id", ce.Context.EventID).
			Str("service-name", serviceName).
			Str("gateway-name", gateway.Name).
			Str("gateway-port", gateway.Port).
			Str("gateway-endpoint", gateway.Endpoint).Msg("dispatching cloudevent to gateway")

		// make an http post request to gateway watcher containing event in request body
		err = toc.postCloudEventToWatcher(ip, gateway.Port, gateway.Endpoint, eventBytes)
		if err != nil {
			toc.log.Error().
				Str("event-source", ce.Context.Source.Host).
				Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).
				Str("sensor-name", gateway.Name).
				Err(err).Msg("failed to send event to watcher")
		} else {
			toc.log.Error().
				Str("event-source", ce.Context.Source.Host).
				Str("event-id", ce.Context.EventID).
				Str("service-name", serviceName).
				Str("sensor-name", gateway.Name).Msg("event dispatched to watcher")
		}
	}
	toc.log.Info().
		Str("event-source", ce.Context.Source.Host).
		Str("event-id", ce.Context.EventID).
		Str("event-source", ce.Context.Source.Host).
		Str("", ce.Context.EventID).Msg("event sent to all watchers")
	return nil
}

// transforms the event into cloudevent specification compliant event
func (toc *tOperationCtx) TransformRequest(w http.ResponseWriter, r *http.Request) {
	toc.log.Info().Msg("transforming incoming request into cloudevent")

	// transform the event
	ce, err := toc.transform(r)
	if err != nil {
		toc.log.Error().Err(err).Msg("failed to transform user event into CloudEvent")
		common.SendErrorResponse(w)
		return
	}

	// dispatch the cloudevent to sensors and gateway watchers for this gateway
	err = toc.dispatchTransformedEvent(ce)
	if err != nil {
		toc.log.Error().Err(err).Str("event-id", ce.Context.EventID).Msg("failed to send the event to watchers")
		common.SendErrorResponse(w)
		return
	}
	common.SendSuccessResponse(w)
}

// ReadinessProbe is probe to check whether server is running or not
func (toc *tOperationCtx) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	toc.log.Info().Msg("transformer is up and running")
	common.SendSuccessResponse(w)
}
