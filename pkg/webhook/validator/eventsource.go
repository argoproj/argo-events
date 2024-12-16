package validator

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsclient "github.com/argoproj/argo-events/pkg/client/clientset/versioned/typed/events/v1alpha1"
	eventsourcecontroller "github.com/argoproj/argo-events/pkg/reconciler/eventsource"
)

type eventsource struct {
	client            kubernetes.Interface
	eventBusClient    eventsclient.EventBusInterface
	eventSourceClient eventsclient.EventSourceInterface
	sensorClient      eventsclient.SensorInterface

	oldes *v1alpha1.EventSource
	newes *v1alpha1.EventSource
}

// NewEventSourceValidator returns a validator for EventSource
func NewEventSourceValidator(client kubernetes.Interface, ebClient eventsclient.EventBusInterface,
	esClient eventsclient.EventSourceInterface, sClient eventsclient.SensorInterface, old, new *v1alpha1.EventSource) Validator {
	return &eventsource{client: client, eventBusClient: ebClient, eventSourceClient: esClient, sensorClient: sClient, oldes: old, newes: new}
}

func (es *eventsource) ValidateCreate(ctx context.Context) *admissionv1.AdmissionResponse {
	if err := eventsourcecontroller.ValidateEventSource(es.newes); err != nil {
		return DeniedResponse("invalid EventSource: %s", err.Error())
	}
	return AllowedResponse()
}

func (es *eventsource) ValidateUpdate(ctx context.Context) *admissionv1.AdmissionResponse {
	if es.oldes.Generation == es.newes.Generation {
		return AllowedResponse()
	}
	return es.ValidateCreate(ctx)
}
