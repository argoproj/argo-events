package validator

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsclient "github.com/argoproj/argo-events/pkg/client/clientset/versioned/typed/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

// Validator is an interface for CRD objects
type Validator interface {
	ValidateCreate(context.Context) *admissionv1.AdmissionResponse
	ValidateUpdate(context.Context) *admissionv1.AdmissionResponse
}

// GetValidator returns a Validator instance
func GetValidator(ctx context.Context, client kubernetes.Interface, aeClient eventsclient.ArgoprojV1alpha1Interface,
	kind metav1.GroupVersionKind, oldBytes []byte, newBytes []byte) (Validator, error) {
	log := logging.FromContext(ctx)
	switch kind.Kind {
	case aev1.EventBusGroupVersionKind.Kind:
		var new *aev1.EventBus
		if len(newBytes) > 0 {
			new = &aev1.EventBus{}
			if err := json.Unmarshal(newBytes, new); err != nil {
				log.Errorf("Could not unmarshal new raw object: %v", err)
				return nil, err
			}
		}
		var old *aev1.EventBus
		if len(oldBytes) > 0 {
			old = &aev1.EventBus{}
			if err := json.Unmarshal(oldBytes, old); err != nil {
				log.Errorf("Could not unmarshal old raw object: %v", err)
				return nil, err
			}
		}
		return NewEventBusValidator(client, aeClient.EventBus(new.Namespace), aeClient.EventSources(new.Namespace), aeClient.Sensors(new.Namespace), old, new), nil
	case aev1.EventSourceGroupVersionKind.Kind:
		var new *aev1.EventSource
		if len(newBytes) > 0 {
			new = &aev1.EventSource{}
			if err := json.Unmarshal(newBytes, new); err != nil {
				log.Errorf("Could not unmarshal new raw object: %v", err)
				return nil, err
			}
		}
		var old *aev1.EventSource
		if len(oldBytes) > 0 {
			old = &aev1.EventSource{}
			if err := json.Unmarshal(oldBytes, old); err != nil {
				log.Errorf("Could not unmarshal old raw object: %v", err)
				return nil, err
			}
		}
		return NewEventSourceValidator(client, aeClient.EventBus(new.Namespace), aeClient.EventSources(new.Namespace), aeClient.Sensors(new.Namespace), old, new), nil
	case aev1.SensorGroupVersionKind.Kind:
		var new *aev1.Sensor
		if len(newBytes) > 0 {
			new = &aev1.Sensor{}
			if err := json.Unmarshal(newBytes, new); err != nil {
				log.Errorf("Could not unmarshal new raw object: %v", err)
				return nil, err
			}
		}
		var old *aev1.Sensor
		if len(oldBytes) > 0 {
			old = &aev1.Sensor{}
			if err := json.Unmarshal(oldBytes, old); err != nil {
				log.Errorf("Could not unmarshal old raw object: %v", err)
				return nil, err
			}
		}
		return NewSensorValidator(client, aeClient.EventBus(new.Namespace), aeClient.EventSources(new.Namespace), aeClient.Sensors(new.Namespace), old, new), nil
	default:
		return nil, fmt.Errorf("unrecognized GVK %v", kind)
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
