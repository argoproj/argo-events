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

func GetDriver(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, eventSourceName string, defaultSubject string, hostname string) (Driver, error) {
	auth, err := eventbus.GetAuth(ctx, eventBusConfig)
	if err != nil {
		return nil, err
	}
	logger := logging.FromContext(ctx)

	var eventBusType apicommon.EventBusType
	if eventBusConfig.NATS != nil {
		eventBusType = apicommon.EventBusNATS
	} else if eventBusConfig.JetStream != nil {
		eventBusType = apicommon.EventBusJetStream
	} else {
		return nil, errors.New("invalid event bus")
	}

	var dvr Driver
	switch eventBusType {
	case apicommon.EventBusNATS:
		dvr = NewNATSStreaming(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, eventSourceName, defaultSubject, auth, logger)
	//case apicommon.EventBusJetStream:
	//	dvr = NewJetstream(eventBusConfig.JetStream.URL, eventSourceName, auth, logger) // don't need to pass in subject because subjects will be derived from dependencies
	default:
		return nil, errors.New("invalid eventbus type")
	}
	return dvr, nil
}
