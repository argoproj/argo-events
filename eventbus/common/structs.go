package common

import (
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

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
