package sourceeventbus

import (
	"context"

	"github.com/argoproj/argo-events/eventbus"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"

	"github.com/argoproj/argo-events/common/logging"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/pkg/errors"
)

type Driver interface {
	Connect(clientID string) (SourceConnection, error)
}

type SourceConnection interface {
	eventbusdriver.Connection

	PublishEvent(ctx context.Context,
		evt eventbusdriver.Event,
		message []byte) error
}

func GetDriver(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, eventSourceName string, defaultSubject string) (Driver, error) {
	auth, err := eventbus.GetAuth(ctx, eventBusConfig)
	if err != nil {
		return nil, err
	}
	if eventSourceName == "" {
		return nil, errors.New("eventSourceName must be specified to create eventbus driver")
	}

	logger := logging.FromContext(ctx)

	logger.Infof("eventBusConfig: %+v", eventBusConfig)

	var eventBusType apicommon.EventBusType
	switch {
	case eventBusConfig.NATS != nil && eventBusConfig.JetStream != nil:
		return nil, errors.New("invalid event bus, NATS and Jetstream shouldn't both be specified")
	case eventBusConfig.NATS != nil:
		eventBusType = apicommon.EventBusNATS
	case eventBusConfig.JetStream != nil:
		eventBusType = apicommon.EventBusJetStream
	default:
		return nil, errors.New("invalid event bus")
	}

	var dvr Driver
	switch eventBusType {
	case apicommon.EventBusNATS:
		if defaultSubject == "" {
			return nil, errors.New("subject must be specified to create NATS Streaming driver")
		}
		dvr = NewNATSStreaming(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, eventSourceName, defaultSubject, auth, logger)
	case apicommon.EventBusJetStream:
		dvr, err = NewJetstream(eventBusConfig.JetStream.URL, eventSourceName, auth, logger) // don't need to pass in subject because subjects will be derived from dependencies
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid eventbus type")
	}
	return dvr, nil
}
