package eventbus

import (
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// ValidateEventBus accepts an EventBus and performs validation against it
func ValidateEventBus(eb *v1alpha1.EventBus) error {
	if eb.Spec.NATS == nil && eb.Spec.JetStream == nil && eb.Spec.Kafka == nil {
		return fmt.Errorf("invalid spec: either \"nats\", \"jetstream\", or \"kafka\" needs to be specified")
	}
	if x := eb.Spec.NATS; x != nil {
		if x.Native != nil && x.Exotic != nil {
			return fmt.Errorf("\"spec.nats.native\" and \"spec.nats.exotic\" can not be defined together")
		}
		if x.Native == nil && x.Exotic == nil {
			return fmt.Errorf("either \"native\" or \"exotic\" must be defined")
		}
		if x.Exotic != nil {
			e := x.Exotic
			if e.ClusterID == nil {
				return fmt.Errorf("\"spec.nats.exotic.clusterID\" is missing")
			}
			if e.URL == "" {
				return fmt.Errorf("\"spec.nats.exotic.url\" is missing")
			}
		}
	}
	if x := eb.Spec.JetStream; x != nil {
		if x.Version == "" {
			return fmt.Errorf("invalid spec: a version for jetstream needs to be specified")
		}
		if x.Replicas != nil && *x.Replicas < 3 {
			return fmt.Errorf("invalid spec: a jetstream eventbus requires at least 3 replicas")
		}
	}
	if x := eb.Spec.Kafka; x != nil {
		if x.Exotic == nil {
			return fmt.Errorf("\"exotic\" must be defined")
		}
		if x.Exotic.URL == "" {
			return fmt.Errorf("\"spec.kafka.exotic.url\" is missing")
		}
	}
	return nil
}
