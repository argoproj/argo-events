package validator

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	sensorcontroller "github.com/argoproj/argo-events/controllers/sensor"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	eventbusclient "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	eventsourceclient "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensorclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
)

type sensor struct {
	client            kubernetes.Interface
	eventBusClient    eventbusclient.Interface
	eventSourceClient eventsourceclient.Interface
	sensorClient      sensorclient.Interface

	oldSensor *sensorv1alpha1.Sensor
	newSensor *sensorv1alpha1.Sensor
}

// NewSensorValidator returns a validator for Sensor
func NewSensorValidator(client kubernetes.Interface, ebClient eventbusclient.Interface,
	esClient eventsourceclient.Interface, sClient sensorclient.Interface, old, new *sensorv1alpha1.Sensor) Validator {
	return &sensor{client: client, eventBusClient: ebClient, eventSourceClient: esClient, sensorClient: sClient, oldSensor: old, newSensor: new}
}

func (s *sensor) ValidateCreate(ctx context.Context) *admissionv1.AdmissionResponse {
	eventBusName := common.DefaultEventBusName
	if len(s.newSensor.Spec.EventBusName) > 0 {
		eventBusName = s.newSensor.Spec.EventBusName
	}
	eventBus, err := s.eventBusClient.ArgoprojV1alpha1().EventBus(s.newSensor.Namespace).Get(ctx, eventBusName, metav1.GetOptions{})
	if err != nil {
		return DeniedResponse(fmt.Sprintf("failed to get EventBus eventBusName=%s; err=%v", eventBusName, err))
	}

	if err := sensorcontroller.ValidateSensor(s.newSensor, eventBus); err != nil {
		return DeniedResponse(err.Error())
	}
	return AllowedResponse()
}

func (s *sensor) ValidateUpdate(ctx context.Context) *admissionv1.AdmissionResponse {
	if s.oldSensor.Generation == s.newSensor.Generation {
		AllowedResponse()
	}
	return s.ValidateCreate(ctx)
}
