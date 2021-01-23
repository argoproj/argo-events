package eventbus

import (
	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// ValidateEventBus accepts an EventBus and performs validation against it
func ValidateEventBus(eb *v1alpha1.EventBus) error {
	if eb.Spec.NATS != nil {
		if eb.Spec.NATS.Native != nil && eb.Spec.NATS.Exotic != nil {
			return errors.New("Native and Exotic can not be defined together")
		}
		if eb.Spec.NATS.Exotic != nil {
			e := eb.Spec.NATS.Exotic
			if e.ClusterID == nil {
				return errors.New("ClusterID is missing")
			}
			if e.URL == "" {
				return errors.New("URL is missing")
			}
		}
	}
	return nil
}
