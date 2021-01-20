package validator

import (
	"context"

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

func (s *sensor) ValidateCreate(ctx context.Context) (bool, string, error) {
	if err := sensorcontroller.ValidateSensor(s.newSensor); err != nil {
		return false, err.Error(), nil
	}
	return true, "", nil
}

func (s *sensor) ValidateUpdate(ctx context.Context) (bool, string, error) {
	if err := sensorcontroller.ValidateSensor(s.newSensor); err != nil {
		return false, err.Error(), nil
	}
	return true, "", nil
}

func (s *sensor) ValidateDelete(ctx context.Context) (bool, string, error) {
	return true, "", nil
}
