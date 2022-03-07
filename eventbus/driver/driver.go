package driver

import (
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// Connection is an interface of event bus driver
type Connection interface {
	Close() error

	IsClosed() bool

	Publish(subject string, data []byte) error

	ClientID() string
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
