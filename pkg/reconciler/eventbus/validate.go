package eventbus

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

// ValidateEventBus accepts an EventBus and performs validation against it
func ValidateEventBus(eb *v1alpha1.EventBus) error {
	if eb.Spec.NATS == nil && eb.Spec.JetStream == nil && eb.Spec.Kafka == nil && eb.Spec.JetStreamExotic == nil {
		return fmt.Errorf("invalid spec: either \"nats\", \"jetstream\", \"jetstreamExotic\", or \"kafka\" needs to be specified")
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
		if x.Replicas != nil && (*x.Replicas == 2 || *x.Replicas <= 0) {
			return fmt.Errorf("invalid spec: a jetstream eventbus requires 1 replica or >= 3 replicas")
		}
		if x.ClientAdvertise != nil {
			if err := validateClientAdvertise(*x.ClientAdvertise); err != nil {
				return fmt.Errorf("invalid spec: \"spec.jetstream.clientAdvertise\" %w", err)
			}
		}
	}
	if x := eb.Spec.Kafka; x != nil {
		if x.URL == "" {
			return fmt.Errorf("\"spec.kafka.url\" is missing")
		}
	}
	if x := eb.Spec.JetStreamExotic; x != nil {
		if x.URL == "" {
			return fmt.Errorf("\"spec.jetstreamExotic.url\" is missing")
		}
	}
	return nil
}

// validateClientAdvertise checks that the value is a "host:port" the NATS server
// config can take as is. It is written into a quoted string of the config, so
// quotes and whitespace would break the whole file.
func validateClientAdvertise(s string) error {
	if strings.ContainsAny(s, " \t\r\n\"'") || strings.Contains(s, "://") {
		return fmt.Errorf("must be in the form \"host:port\", got %q", s)
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil || host == "" {
		return fmt.Errorf("must be in the form \"host:port\", got %q", s)
	}
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("has an invalid port %q", port)
	}
	return nil
}
