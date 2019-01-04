package kafka

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

// ValidateEventSource validates the gateway event source
func (kce *KafkaConfigExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	kafkaConfig, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}
	if kafkaConfig == nil {
		return v, fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidEventSource)
	}
	if kafkaConfig.URL == "" {
		return v, fmt.Errorf("%+v, url must be specified", gateways.ErrInvalidEventSource)
	}
	if kafkaConfig.Topic == "" {
		return v, fmt.Errorf("%+v, topic must be specified", gateways.ErrInvalidEventSource)
	}
	if kafkaConfig.Partition == "" {
		return v, fmt.Errorf("%+v, partition must be specified", gateways.ErrInvalidEventSource)
	}
	return v, nil
}
