package validator

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common/logging"
	eventbuspkg "github.com/argoproj/argo-events/pkg/apis/eventbus"
	eventbusv1alphal1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcepkg "github.com/argoproj/argo-events/pkg/apis/eventsource"
	sensorpkg "github.com/argoproj/argo-events/pkg/apis/sensor"
)

// Validator is an interface for CRD objects
type Validator interface {
	ValidateCreate(context.Context) (bool, string, error)
	ValidateUpdate(context.Context) (bool, string, error)
}

// GetValidator returns a Validator
func GetValidator(ctx context.Context, client kubernetes.Interface, kind metav1.GroupVersionKind, oldBytes []byte, newBytes []byte) (Validator, error) {
	log := logging.FromContext(ctx)
	switch kind.Kind {
	case eventbuspkg.Kind:
		new := &eventbusv1alphal1.EventBus{}
		if err := json.Unmarshal(newBytes, new); err != nil {
			log.Errorf("Could not unmarshal new raw object: %v", err)
			return nil, err
		}
		var old *eventbusv1alphal1.EventBus
		if len(oldBytes) > 0 {
			old = &eventbusv1alphal1.EventBus{}
			if err := json.Unmarshal(oldBytes, old); err != nil {
				log.Errorf("Could not unmarshal old raw object: %v", err)
				return nil, err
			}
		}
		return NewEventBusValidator(client, old, new), nil
	case eventsourcepkg.Kind:
		return nil, nil
	case sensorpkg.Kind:
		return nil, nil
	default:
		return nil, errors.Errorf("unrecognized GVK %v", kind)
	}
}
