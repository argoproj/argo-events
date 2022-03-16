package common

import (
	"context"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type Connection interface {
	Close() error

	IsClosed() bool

	ClientID() string
}

type EventSourceConnection interface {
	Connection

	Publish(ctx context.Context, evt Event, message []byte) error
}

type TriggerConnection interface {
	Connection

	Subscribe(ctx context.Context,
		closeCh <-chan struct{},
		resetConditionsCh <-chan struct{},
		lastResetTime time.Time,
		transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
		filter func(string, cloudevents.Event) bool,
		action func(map[string]cloudevents.Event),
		defaultSubject *string) error
}

type EventSourceDriver interface {
	Connect(clientID string) (EventSourceConnection, error)
}

type SensorDriver interface {
	Connect(triggerName string, dependencyExpression string, deps []Dependency) (TriggerConnection, error)
}
