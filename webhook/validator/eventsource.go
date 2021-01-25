package validator

import (
	"context"

	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
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
	log := logging.FromContext(ctx)
	if err := eventsourcecontroller.ValidateEventSource(es.newes); err != nil {
		return DeniedResponse(err.Error())
	}
	ebName := es.newes.Spec.EventBusName
	if ebName == "" {
		ebName = common.DefaultEventBusName
	}
	eb, err := es.eventBusClient.ArgoprojV1alpha1().EventBus(es.newes.Namespace).Get(ctx, ebName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return DeniedResponse("An EventBus named \"%s\" needs to be created first", ebName)
		}
		log.Errorw("failed to retrieve EventBus", zap.Error(err))
		return DeniedResponse("Failed to retrieve the EventBus, %s", err.Error())
	}
	if !eb.Status.IsReady() {
		return DeniedResponse("EventBus \"%s\" is not in a good shape", ebName)
	}
	return AllowedResponse()
}

func (es *eventsource) ValidateUpdate(ctx context.Context) *admissionv1.AdmissionResponse {
	if es.oldes.Generation == es.newes.Generation {
		return AllowedResponse()
	}
	return es.ValidateCreate(ctx)
}

func (es *eventsource) ValidateDelete(ctx context.Context) *admissionv1.AdmissionResponse {
	return AllowedResponse()
}
