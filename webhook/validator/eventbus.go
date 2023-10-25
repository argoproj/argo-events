package validator

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	eventbuscontroller "github.com/argoproj/argo-events/controllers/eventbus"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventbusclient "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned"
	eventsourceclient "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensorclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
)

type eventbus struct {
	client            kubernetes.Interface
	eventBusClient    eventbusclient.Interface
	eventSourceClient eventsourceclient.Interface
	sensorClient      sensorclient.Interface

	oldeb *eventbusv1alpha1.EventBus
	neweb *eventbusv1alpha1.EventBus
}

// NewEventBusValidator returns a validator for EventBus
func NewEventBusValidator(client kubernetes.Interface, ebClient eventbusclient.Interface,
	esClient eventsourceclient.Interface, sClient sensorclient.Interface, old, new *eventbusv1alpha1.EventBus) Validator {
	return &eventbus{client: client, eventBusClient: ebClient, eventSourceClient: esClient, sensorClient: sClient, oldeb: old, neweb: new}
}

func (eb *eventbus) ValidateCreate(ctx context.Context) *admissionv1.AdmissionResponse {
	if err := eventbuscontroller.ValidateEventBus(eb.neweb); err != nil {
		return DeniedResponse(err.Error())
	}

	return AllowedResponse()
}

func (eb *eventbus) ValidateUpdate(ctx context.Context) *admissionv1.AdmissionResponse {
	if eb.oldeb.Generation == eb.neweb.Generation {
		return AllowedResponse()
	}
	if err := eventbuscontroller.ValidateEventBus(eb.neweb); err != nil {
		return DeniedResponse(err.Error())
	}
	switch {
	case eb.neweb.Spec.NATS != nil:
		if eb.oldeb.Spec.NATS == nil {
			return DeniedResponse("Can not change event bus implementation")
		}
		oldNats := eb.oldeb.Spec.NATS
		newNats := eb.neweb.Spec.NATS
		if newNats.Native != nil {
			if oldNats.Native == nil {
				return DeniedResponse("Can not change NATS event bus implementation from exotic to native")
			}
			if authChanged(oldNats.Native.Auth, newNats.Native.Auth) {
				return DeniedResponse("\"spec.nats.native.auth\" is immutable, can not be updated")
			}
		} else if newNats.Exotic != nil {
			if oldNats.Exotic == nil {
				return DeniedResponse("Can not change NATS event bus implementation from native to exotic")
			}
			if authChanged(oldNats.Exotic.Auth, newNats.Exotic.Auth) {
				return DeniedResponse("\"spec.nats.exotic.auth\" is immutable, can not be updated")
			}
		}
	case eb.neweb.Spec.JetStream != nil:
		if eb.oldeb.Spec.JetStream == nil {
			return DeniedResponse("Can not change event bus implementation")
		}
		oldJs := eb.oldeb.Spec.JetStream
		newJs := eb.neweb.Spec.JetStream
		if (oldJs.StreamConfig == nil && newJs.StreamConfig != nil) ||
			(oldJs.StreamConfig != nil && newJs.StreamConfig == nil) {
			return DeniedResponse("\"spec.jetstream.streamConfig\" is immutable, can not be updated")
		}
		if oldJs.StreamConfig != nil && newJs.StreamConfig != nil && *oldJs.StreamConfig != *newJs.StreamConfig {
			return DeniedResponse("\"spec.jetstream.streamConfig\" is immutable, can not be updated, old value='%s', new value='%s'", *oldJs.StreamConfig, *newJs.StreamConfig)
		}
	case eb.neweb.Spec.JetStreamExotic != nil:
		if eb.oldeb.Spec.JetStreamExotic == nil {
			return DeniedResponse("Can not change event bus implementation")
		}
	}

	return AllowedResponse()
}

func authChanged(old, new *eventbusv1alpha1.AuthStrategy) bool {
	if old == nil && new == nil {
		return false
	}
	if old == nil {
		return *new != eventbusv1alpha1.AuthStrategyNone
	}
	if new == nil {
		return *old != eventbusv1alpha1.AuthStrategyNone
	}
	return *new != *old
}
