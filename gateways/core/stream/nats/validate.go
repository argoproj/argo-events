package nats

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

// ValidateEventSource validates gateway event source
func (ese *NatsEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	n, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}
	if n == nil {
		return v, fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidEventSource)
	}
	if n.URL == "" {
		return v, fmt.Errorf("%+v, url must be specified", gateways.ErrInvalidEventSource)
	}
	if n.Subject == "" {
		return v, fmt.Errorf("%+v, subject must be specified", gateways.ErrInvalidEventSource)
	}
	return v, nil
}
