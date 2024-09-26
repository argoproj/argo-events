package common

// EventBusError is a particular EventBus related error.
type EventBusError struct {
	err error
}

func (e *EventBusError) Error() string {
	return e.err.Error()
}

// NewEventBusError returns an EventBusError.
func NewEventBusError(err error) error {
	return &EventBusError{err: err}
}
