package eventbus

import (
	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// ValidateEventBus accepts an EventBus and performs validation against it
func ValidateEventBus(eb *v1alpha1.EventBus) error {
	if eb.Spec.NATS == nil {
		return errors.New("invalid spec: missing \"spec.nats\"")
	}
	if eb.Spec.NATS.Native != nil && eb.Spec.NATS.Exotic != nil {
		return errors.New("\"spec.nats.native\" and \"spec.nats.exotic\" can not be defined together")
	}
	if eb.Spec.NATS.Native == nil && eb.Spec.NATS.Exotic == nil {
		return errors.New("either \"native\" or \"exotic\" must be defined")
	}
	if eb.Spec.NATS.Exotic != nil {
		e := eb.Spec.NATS.Exotic
		if e.ClusterID == nil {
			return errors.New("\"spec.nats.exotic.clusterID\" is missing")
		}
		if e.URL == "" {
			return errors.New("\"spec.nats.exotic.url\" is missing")
		}
	}
	return nil
}
