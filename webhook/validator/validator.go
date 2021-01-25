package validator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common/logging"
	eventbuspkg "github.com/argoproj/argo-events/pkg/apis/eventbus"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/apis/eventsource"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorpkg "github.com/argoproj/argo-events/pkg/apis/sensor"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	eventbusclient "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned"
	eventsourceclient "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensorclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
)

// Validator is an interface for CRD objects
type Validator interface {
	ValidateCreate(context.Context) *admissionv1.AdmissionResponse
	ValidateUpdate(context.Context) *admissionv1.AdmissionResponse
	ValidateDelete(context.Context) *admissionv1.AdmissionResponse
}

// GetValidator returns a Validator instance
func GetValidator(ctx context.Context, client kubernetes.Interface, ebClient eventbusclient.Interface,
	esClient eventsourceclient.Interface, sensorClient sensorclient.Interface,
	kind metav1.GroupVersionKind, oldBytes []byte, newBytes []byte) (Validator, error) {
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
		return NewEventBusValidator(client, ebClient, esClient, sensorClient, old, new), nil
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
		return NewEventSourceValidator(client, ebClient, esClient, sensorClient, old, new), nil
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
		return NewSensorValidator(client, ebClient, esClient, sensorClient, old, new), nil
	default:
		return nil, errors.Errorf("unrecognized GVK %v", kind)
	}
}

// DeniedResponse constructs a denied AdmissionResonse
func DeniedResponse(reason string, args ...interface{}) *admissionv1.AdmissionResponse {
	result := apierrors.NewBadRequest(fmt.Sprintf(reason, args...)).Status()
	return &admissionv1.AdmissionResponse{
		Result:  &result,
		Allowed: false,
	}
}

// AllowedResponse constructs an allowed AdmissionResonse
func AllowedResponse() *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
