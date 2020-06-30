package driver

import (
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Driver is an interface for event bus
type Driver interface {
	Connect() (Connection, error)

	// SubscribeEventSources is used to subscribe multiple event source dependencies
	// Parameter - dependencyExpr, example: "(dep1 || dep2) && dep3"
	// Parameter - depencencies, array of dependencies information
	// Parameter - filter, a function used to filter the message
	// Parameter - action, a function to be triggered after all conditions meet
	SubscribeEventSources(conn Connection, dependencyExpr string, depencencies []Dependency, filter func(string, cloudevents.Event) bool, action func(map[string]cloudevents.Event)) error

	// Publish a message
	Publish(conn Connection, message []byte) error
}

// Connection is an interface of event bus driver
type Connection interface {
	Close() error

	IsClosed() bool

	Publish(subject string, data []byte) error
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

// Dependency is a struct for dependency info of a sensor
type Dependency struct {
	Name            string
	EventSourceName string
	EventName       string
}
