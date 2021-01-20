package validator

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
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
	fmt.Printf("CREATE - old: %v\n", eb.oldeb)
	if eb.neweb.Spec.NATS != nil && eb.neweb.Spec.NATS.Native != nil && eb.neweb.Spec.NATS.Native.Replicas < 3 {
		return false, "Replicas must be no less than 3", nil
	}
	return true, "", nil
}

func (eb *eventbus) ValidateUpdate(ctx context.Context) (bool, string, error) {
	fmt.Printf("UPDATE - old: %v\n", eb.oldeb)
	fmt.Printf("UPDATE - new: %v\n", eb.neweb)
	if equality.Semantic.DeepEqual(eb.oldeb.Spec, eb.neweb.Spec) {
		return true, "", nil
	}

	if eb.neweb.Spec.NATS != nil && eb.neweb.Spec.NATS.Native != nil && eb.neweb.Spec.NATS.Native.Replicas < 3 {
		return false, "Replicas must be no less than 3", nil
	}
	return true, "", nil
}

func (eb *eventbus) ValidateDelete(ctx context.Context) (bool, string, error) {
	fmt.Printf("Delete - new: %v\n", eb.neweb)
	fmt.Printf("Delete - old: %v\n", eb.oldeb)
	return true, "", nil
}
