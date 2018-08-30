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
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"time"
)

type tConfig struct {
	// EventType is type of the event
	EventType string
	// EventTypeVersion is the version of the `eventType`
	EventTypeVersion string
	// Source of the event
	EventSource string
	// Sensors to dispatch the event to
	Sensors []string
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

func NewTransformerConfig(eventType string, eventTypeVersion string, eventSource string, sensors []string) *tConfig {
	return &tConfig{
		EventType:        eventType,
		EventTypeVersion: eventTypeVersion,
		EventSource:      eventSource,
		Sensors:          sensors,
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
				Host: toc.Config.EventSource,
			},
		},
		Payload: payload,
	}
	return ce, nil
}

// dispatches the event to configured sensor
func (toc *tOperationCtx) dispatchTransformedEvent(ce *sv1alpha.Event) error {
	for _, sensor := range toc.Config.Sensors {
		sensorService, err := toc.kubeClientset.CoreV1().Services(toc.Namespace).Get(common.DefaultSensorServiceName(sensor), metav1.GetOptions{})
		if err != nil {
			toc.log.Error().Str("sensor-service-name", sensorService.Name).Err(err).Msg("failed to connect to sensor service")
			return err
		}

		// the sensor service exposed as intra cluster service
		if sensorService.Spec.ClusterIP == "" {
			toc.log.Error().Str("sensor-service", sensorService.Name).Err(err).Msg("failed to connect to sensor service")
			return err
		}

		// get the byte array from cloudevent to dispatch to sensor service
		eventBytes, err := json.Marshal(ce)
		if err != nil {
			toc.log.Error().Err(err).Msg("failed to get byte array from cloudevent")
			return err
		}

		// dispatch the cloudevent
		toc.log.Info().Str("sensor-service-name", sensorService.Name).Str("sensor-name", sensor).Msg("dispatching cloudevent to sensor")
		req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d", sensorService.Spec.ClusterIP, common.SensorServicePort), bytes.NewBuffer(eventBytes))
		if err != nil {
			toc.log.Error().Msg("unable to create a http request")
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		_, err = client.Do(req)
		if err != nil {
			return err
		}
	}
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
