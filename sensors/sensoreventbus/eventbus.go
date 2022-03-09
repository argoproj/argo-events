package sensoreventbus

import (
	"context"
	"time"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
)

type Driver interface {
	Connect(triggerName string, dependencyExpression string, deps []Dependency) (TriggerConnection, error)
}

type TriggerConnection interface {
	eventbusdriver.Connection

	Subscribe(ctx context.Context,
		closeCh <-chan struct{},
		resetConditionsCh <-chan struct{},
		lastResetTime time.Time,
		transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
		filter func(string, cloudevents.Event) bool,
		action func(map[string]cloudevents.Event),
		defaultSubject *string) error
}

// Dependency is a struct for dependency info of a sensor
type Dependency struct {
	Name            string
	EventSourceName string
	EventName       string
}

func GetSensorDriver(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, sensorSpec *v1alpha1.Sensor) (Driver, error) {
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
		dvr = NewNATSStreaming(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, sensorSpec.Name, auth, logger)
	//case apicommon.EventBusJetstream:
	//	dvr = NewJetstream(eventBusConfig.Jetstream.URL, sensorSpec, auth, logger) // don't need to pass in subject because subjects will be derived from dependencies
	default:
		return nil, errors.New("invalid eventbus type")
	}
	return dvr, nil
}
