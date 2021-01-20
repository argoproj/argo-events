package validator

import (
	"context"

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

func (es *eventsource) ValidateCreate(ctx context.Context) (bool, string, error) {
	if err := eventsourcecontroller.ValidateEventSource(es.newes); err != nil {
		return false, err.Error(), nil
	}
	return true, "", nil
}

func (es *eventsource) ValidateUpdate(ctx context.Context) (bool, string, error) {
	if err := eventsourcecontroller.ValidateEventSource(es.newes); err != nil {
		return false, err.Error(), nil
	}
	return true, "", nil
}

func (es *eventsource) ValidateDelete(ctx context.Context) (bool, string, error) {
	return true, "", nil
}
