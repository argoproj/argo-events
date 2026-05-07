package validator

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsclient "github.com/argoproj/argo-events/pkg/client/clientset/versioned/typed/events/v1alpha1"
	sensorcontroller "github.com/argoproj/argo-events/pkg/reconciler/sensor"
)

type sensor struct {
	client    kubernetes.Interface
	aeClient  eventsclient.ArgoprojV1alpha1Interface
	// eventBusClient is kept for callers that pass a pre-scoped client (e.g. tests).
	// When aeClient is set it takes precedence, allowing cross-namespace resolution.
	eventBusClient    eventsclient.EventBusInterface
	eventSourceClient eventsclient.EventSourceInterface
	sensorClient      eventsclient.SensorInterface

	oldSensor *v1alpha1.Sensor
	newSensor *v1alpha1.Sensor
}

// NewSensorValidator returns a validator for Sensor.
// aeClient, when non-nil, is used to resolve the EventBus in the namespace specified by
// EventBusNamespace (or the Sensor's own namespace when the field is unset). ebClient is
// used as a fallback when aeClient is nil (e.g. in unit tests that pre-scope the client).
func NewSensorValidator(client kubernetes.Interface, aeClient eventsclient.ArgoprojV1alpha1Interface,
	ebClient eventsclient.EventBusInterface, esClient eventsclient.EventSourceInterface,
	sClient eventsclient.SensorInterface, old, new *v1alpha1.Sensor) Validator {
	return &sensor{
		client:            client,
		aeClient:          aeClient,
		eventBusClient:    ebClient,
		eventSourceClient: esClient,
		sensorClient:      sClient,
		oldSensor:         old,
		newSensor:         new,
	}
}

func (s *sensor) ValidateCreate(ctx context.Context) *admissionv1.AdmissionResponse {
	eventBusName := v1alpha1.DefaultEventBusName
	if len(s.newSensor.Spec.EventBusName) > 0 {
		eventBusName = s.newSensor.Spec.EventBusName
	}

	// Resolve the EventBus client, preferring the full aeClient so we can look up
	// an EventBus in a different namespace when EventBusNamespace is set.
	var ebClient eventsclient.EventBusInterface
	if s.aeClient != nil {
		ebNamespace := s.newSensor.Spec.GetEventBusNamespace(s.newSensor.Namespace)
		ebClient = s.aeClient.EventBus(ebNamespace)
	} else {
		ebClient = s.eventBusClient
	}

	if ebClient == nil {
		return DeniedResponse("invalid EventBus: eventBusClient is nil")
	}
	eventBus, err := ebClient.Get(ctx, eventBusName, metav1.GetOptions{})
	if err != nil {
		return DeniedResponse("failed to get EventBus eventBusName=%s; err=%v", eventBusName, err)
	}

	if err := sensorcontroller.ValidateSensor(s.newSensor, eventBus); err != nil {
		return DeniedResponse("invalid Sensor: %s", err.Error())
	}
	return AllowedResponse()
}

func (s *sensor) ValidateUpdate(ctx context.Context) *admissionv1.AdmissionResponse {
	if s.oldSensor.Generation == s.newSensor.Generation {
		return AllowedResponse()
	}
	return s.ValidateCreate(ctx)
}
