package validator

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	eventsourcecontroller "github.com/argoproj/argo-events/controllers/eventsource"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

type eventsource struct {
	client kubernetes.Interface
	oldes  *eventsourcev1alpha1.EventSource
	newes  *eventsourcev1alpha1.EventSource
}

// NewEventSourceValidator returns a validator for EventSource
func NewEventSourceValidator(client kubernetes.Interface, old, new *eventsourcev1alpha1.EventSource) Validator {
	return &eventsource{client: client, oldes: old, newes: new}
}

func (es *eventsource) ValidateCreate(ctx context.Context) *admissionv1.AdmissionResponse {
	if err := eventsourcecontroller.ValidateEventSource(es.newes); err != nil {
		return DeniedResponse(err.Error())
	}
	return AllowedResponse()
}

func (es *eventsource) ValidateUpdate(ctx context.Context) *admissionv1.AdmissionResponse {
	if es.oldes.Generation == es.newes.Generation {
		return AllowedResponse()
	}
	if err := eventsourcecontroller.ValidateEventSource(es.newes); err != nil {
		return DeniedResponse(err.Error())
	}
	return AllowedResponse()
}

func (es *eventsource) ValidateDelete(ctx context.Context) *admissionv1.AdmissionResponse {
	return AllowedResponse()
}
