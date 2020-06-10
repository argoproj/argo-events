package driver

import (
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// Driver is an interface for event bus
type Driver interface {
	Connect() error

	Disconnect() error

	Reconnect() error

	// Subscribe to a subject
	Subscribe(subject string, action func(message []byte)) error

	// Subscribe to multiple subjects
	// Parameter - subjectExpr, example: "(subject1 || subject2) && subject3"
	// Parameter - filter, a function used to filter the message
	// Parameter - action, a function to be triggered after all conditions meet
	SubscribeSubjects(subjectExpr string, filter func(subject string, message []byte) bool, action func(subjectMessages map[string][]byte)) error

	// Publish a message to a subject
	Publish(subject string, message []byte) error
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
