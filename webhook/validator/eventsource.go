package validator

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	eventsourcecontroller "github.com/argoproj/argo-events/controllers/eventsource"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	eventbusclient "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned"
	eventsourceclient "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensorclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
)

type eventsource struct {
	client            kubernetes.Interface
	eventBusClient    eventbusclient.Interface
	eventSourceClient eventsourceclient.Interface
	sensorClient      sensorclient.Interface

	oldes *eventsourcev1alpha1.EventSource
	newes *eventsourcev1alpha1.EventSource
}

// NewEventSourceValidator returns a validator for EventSource
func NewEventSourceValidator(client kubernetes.Interface, ebClient eventbusclient.Interface,
	esClient eventsourceclient.Interface, sClient sensorclient.Interface, old, new *eventsourcev1alpha1.EventSource) Validator {
	return &eventsource{client: client, eventBusClient: ebClient, eventSourceClient: esClient, sensorClient: sClient, oldes: old, newes: new}
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
	return es.ValidateCreate(ctx)
}
