package validator

import (
	"context"

	"k8s.io/client-go/kubernetes"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

type eventbus struct {
	client kubernetes.Interface
	oldeb  *eventbusv1alpha1.EventBus
	neweb  *eventbusv1alpha1.EventBus
}

// NewEventBusValidator returns a validator for EventBus
func NewEventBusValidator(client kubernetes.Interface, old, new *eventbusv1alpha1.EventBus) Validator {
	return &eventbus{client: client, oldeb: old, neweb: new}
}

func (eb *eventbus) ValidateCreate(ctx context.Context) (bool, string, error) {
	if eb.neweb.Spec.NATS != nil && eb.neweb.Spec.NATS.Native != nil && eb.neweb.Spec.NATS.Native.Replicas < 3 {
		return false, "Replicas must be no less than 3", nil
	}
	return true, "", nil
}

func (eb *eventbus) ValidateUpdate(ctx context.Context) (bool, string, error) {
	return true, "", nil
}
