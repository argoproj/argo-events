package validator

import (
	"context"
	"os"

	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventbuscontroller "github.com/argoproj/argo-events/controllers/eventbus"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourceclient "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensorclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
)

type eventbus struct {
	client       kubernetes.Interface
	esClient     *eventsourceclient.Clientset
	sensorClient *sensorclient.Clientset

	oldeb *eventbusv1alpha1.EventBus
	neweb *eventbusv1alpha1.EventBus
}

// NewEventBusValidator returns a validator for EventBus
func NewEventBusValidator(client kubernetes.Interface, old, new *eventbusv1alpha1.EventBus) Validator {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	esClient := eventsourceclient.NewForConfigOrDie(restConfig)
	sensorClient := sensorclient.NewForConfigOrDie(restConfig)
	return &eventbus{client: client, oldeb: old, neweb: new, esClient: esClient, sensorClient: sensorClient}
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
	if eb.neweb.Spec.NATS != nil {
		if eb.oldeb.Spec.NATS == nil {
			return DeniedResponse("Can not change event bus implmementation")
		}
		oldNats := eb.oldeb.Spec.NATS
		newNats := eb.neweb.Spec.NATS
		if newNats.Native != nil {
			if oldNats.Native == nil {
				return DeniedResponse("Can not change NATS event bus implmementation from exotic to native")
			}
			if authChanged(oldNats.Native.Auth, newNats.Native.Auth) {
				return DeniedResponse("Auth strategy is not allowed to be updated")
			}
		} else if newNats.Exotic != nil {
			if oldNats.Exotic == nil {
				return DeniedResponse("Can not change NATS event bus implmementation from native to exotic")
			}
			if authChanged(oldNats.Exotic.Auth, newNats.Exotic.Auth) {
				return DeniedResponse("Auth strategy is not allowed to be updated")
			}
		}
	}
	return AllowedResponse()
}

func (eb *eventbus) ValidateDelete(ctx context.Context) *admissionv1.AdmissionResponse {
	log := logging.FromContext(ctx)
	linkedEventSources, err := eb.linkedEventSources(ctx, eb.oldeb.Namespace, eb.oldeb.Name)
	if err != nil {
		log.Errorw("failed to query linked EventSources", zap.Error(err))
		return DeniedResponse("Failed to query linked EventSources: %s", err.Error())
	}
	if linkedEventSources > 0 {
		return DeniedResponse("Can not delete the EventBus with %v EventSources linked", linkedEventSources)
	}
	linkedSensors, err := eb.linkedSensors(ctx, eb.oldeb.Namespace, eb.oldeb.Name)
	if err != nil {
		log.Errorw("failed to query linked Sensors", zap.Error(err))
		return DeniedResponse("Failed to query linked Sensors: %s", err.Error())
	}
	if linkedSensors > 0 {
		return DeniedResponse("Can not delete the EventBus with %v Sensors linked", linkedSensors)
	}
	return AllowedResponse()
}

func (eb *eventbus) linkedEventSources(ctx context.Context, namespace, eventBusName string) (int, error) {
	esList, err := eb.esClient.ArgoprojV1alpha1().EventSources(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	result := 0
	for _, es := range esList.Items {
		ebName := es.Spec.EventBusName
		if ebName == "" {
			ebName = "default"
		}
		if ebName == eventBusName {
			result++
		}
	}
	return result, nil
}

func (eb *eventbus) linkedSensors(ctx context.Context, namespace, eventBusName string) (int, error) {
	sList, err := eb.sensorClient.ArgoprojV1alpha1().Sensors(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}
	result := 0
	for _, s := range sList.Items {
		sName := s.Spec.EventBusName
		if sName == "" {
			sName = "default"
		}
		if sName == eventBusName {
			result++
		}
	}
	return result, nil
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
