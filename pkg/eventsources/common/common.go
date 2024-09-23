package common

import "github.com/cloudevents/sdk-go/v2/event"

type Option func(*event.Event) error

// Option to set different ID for event
func WithID(id string) Option {
	return func(e *event.Event) error {
		e.SetID(id)
		return nil
	}
}
