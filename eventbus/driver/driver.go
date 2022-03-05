package driver

import (
	"context"
	"time"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Driver is an interface for event bus
type SourceEBDriver interface {
	Connect() (SourceConnection, error)

	// Publish a message related to an event (not sure if we need this or not)
	//Publish(conn Connection, message []byte, event Event) error
}

type SensorEBDriver interface {
	Connect() (TriggerConnection, error)
}

// Connection is an interface of event bus driver
type Connection interface {
	Close() error

	IsClosed() bool

	Publish(subject string, data []byte) error
}

type SourceConnection interface {
	Connection
}

type TriggerConnection interface {
	Connection

	Subscribe(ctx context.Context,
		group string,
		//sensorName string,
		//triggerName string,
		closeCh <-chan struct{},
		resetConditionsCh <-chan struct{},
		lastResetTime time.Time,
		dependencyExpr string,
		dependencies []Dependency,
		transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
		filter func(string, cloudevents.Event) bool,
		action func(map[string]cloudevents.Event)) error
}

// Auth contains the auth infor for event bus
type Auth struct {
	Strategy    eventbusv1alpha1.AuthStrategy
	Crendential *AuthCredential
}

// AuthCredential host the credential info
type AuthCredential struct {
	Token    string
	Username string
	Password string
}

type Event struct {
	EventSourceName string
	EventName       string
}

// Dependency is a struct for dependency info of a sensor
type Dependency struct {
	Name            string
	EventSourceName string
	EventName       string
}
