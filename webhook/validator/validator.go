package validator

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common/logging"
	eventbuspkg "github.com/argoproj/argo-events/pkg/apis/eventbus"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/apis/eventsource"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorpkg "github.com/argoproj/argo-events/pkg/apis/sensor"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// Validator is an interface for CRD objects
type Validator interface {
	ValidateCreate(context.Context) (bool, string, error)
	ValidateUpdate(context.Context) (bool, string, error)
	ValidateDelete(context.Context) (bool, string, error)
}

// GetValidator returns a Validator
func GetValidator(ctx context.Context, client kubernetes.Interface, kind metav1.GroupVersionKind, oldBytes []byte, newBytes []byte) (Validator, error) {
	log := logging.FromContext(ctx)
	switch kind.Kind {
	case eventbuspkg.Kind:
		var new *eventbusv1alpha1.EventBus
		if len(newBytes) > 0 {
			new = &eventbusv1alpha1.EventBus{}
			if err := json.Unmarshal(newBytes, new); err != nil {
				log.Errorf("Could not unmarshal new raw object: %v", err)
				return nil, err
			}
		}
		var old *eventbusv1alpha1.EventBus
		if len(oldBytes) > 0 {
			old = &eventbusv1alpha1.EventBus{}
			if err := json.Unmarshal(oldBytes, old); err != nil {
				log.Errorf("Could not unmarshal old raw object: %v", err)
				return nil, err
			}
		}
		return NewEventBusValidator(client, old, new), nil
	case eventsourcepkg.Kind:
		var new *eventsourcev1alpha1.EventSource
		if len(newBytes) > 0 {
			new = &eventsourcev1alpha1.EventSource{}
			if err := json.Unmarshal(newBytes, new); err != nil {
				log.Errorf("Could not unmarshal new raw object: %v", err)
				return nil, err
			}
		}
		var old *eventsourcev1alpha1.EventSource
		if len(oldBytes) > 0 {
			old = &eventsourcev1alpha1.EventSource{}
			if err := json.Unmarshal(oldBytes, old); err != nil {
				log.Errorf("Could not unmarshal old raw object: %v", err)
				return nil, err
			}
		}
		return NewEventSourceValidator(client, old, new), nil
	case sensorpkg.Kind:
		var new *sensorv1alpha1.Sensor
		if len(newBytes) > 0 {
			new = &sensorv1alpha1.Sensor{}
			if err := json.Unmarshal(newBytes, new); err != nil {
				log.Errorf("Could not unmarshal new raw object: %v", err)
				return nil, err
			}
		}
		var old *sensorv1alpha1.Sensor
		if len(oldBytes) > 0 {
			old = &sensorv1alpha1.Sensor{}
			if err := json.Unmarshal(oldBytes, old); err != nil {
				log.Errorf("Could not unmarshal old raw object: %v", err)
				return nil, err
			}
		}
		return NewSensorValidator(client, old, new), nil
	default:
		return nil, errors.Errorf("unrecognized GVK %v", kind)
	}
}
