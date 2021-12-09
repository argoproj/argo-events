package common

import "github.com/cloudevents/sdk-go/v2/event"

type Options func(*event.Event) error

// Option to set different ID for event
func WithID(id string) Options {
	return func(e *event.Event) error {
		e.SetID(id)
		return nil
	}
}
