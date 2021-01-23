package validator

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	sensorcontroller "github.com/argoproj/argo-events/controllers/sensor"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

type sensor struct {
	client    kubernetes.Interface
	oldSensor *sensorv1alpha1.Sensor
	newSensor *sensorv1alpha1.Sensor
}

// NewSensorValidator returns a validator for Sensor
func NewSensorValidator(client kubernetes.Interface, old, new *sensorv1alpha1.Sensor) Validator {
	return &sensor{client: client, oldSensor: old, newSensor: new}
}

func (s *sensor) ValidateCreate(ctx context.Context) *admissionv1.AdmissionResponse {
	if s.oldSensor.Generation == s.newSensor.Generation {
		AllowedResponse()
	}
	if err := sensorcontroller.ValidateSensor(s.newSensor); err != nil {
		return DeniedResponse(err.Error())
	}
	return AllowedResponse()
}

func (s *sensor) ValidateUpdate(ctx context.Context) *admissionv1.AdmissionResponse {
	if err := sensorcontroller.ValidateSensor(s.newSensor); err != nil {
		return DeniedResponse(err.Error())
	}
	return AllowedResponse()
}

func (s *sensor) ValidateDelete(ctx context.Context) *admissionv1.AdmissionResponse {
	return AllowedResponse()
}
