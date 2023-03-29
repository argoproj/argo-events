package common

import (
	"context"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type Connection interface {
	Close() error

	IsClosed() bool
}

type EventSourceConnection interface {
	Connection

	Publish(ctx context.Context, msg Message) error
}

type TriggerConnection interface {
	Connection

	fmt.Stringer // need to implement String()

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
	Initialize() error
	Connect(clientID string) (EventSourceConnection, error)
}

type SensorDriver interface {
	Initialize() error
	Connect(ctx context.Context,
		triggerName string,
		dependencyExpression string,
		deps []Dependency,
		atLeastOnce bool) (TriggerConnection, error)
}
