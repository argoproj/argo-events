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
	client            kubernetes.Interface
	eventBusClient    eventsclient.EventBusInterface
	eventSourceClient eventsclient.EventSourceInterface
	sensorClient      eventsclient.SensorInterface

	oldSensor *v1alpha1.Sensor
	newSensor *v1alpha1.Sensor
}

// NewSensorValidator returns a validator for Sensor
func NewSensorValidator(client kubernetes.Interface, ebClient eventsclient.EventBusInterface,
	esClient eventsclient.EventSourceInterface, sClient eventsclient.SensorInterface, old, new *v1alpha1.Sensor) Validator {
	return &sensor{client: client, eventBusClient: ebClient, eventSourceClient: esClient, sensorClient: sClient, oldSensor: old, newSensor: new}
}

func (s *sensor) ValidateCreate(ctx context.Context) *admissionv1.AdmissionResponse {
	eventBusName := v1alpha1.DefaultEventBusName
	if len(s.newSensor.Spec.EventBusName) > 0 {
		eventBusName = s.newSensor.Spec.EventBusName
	}
	if s.eventBusClient == nil {
		return DeniedResponse("invalid EventBus: eventBusClient is nil")
	}
	eventBus, err := s.eventBusClient.Get(ctx, eventBusName, metav1.GetOptions{})
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
