package generic

import (
	"context"
	"fmt"
)

func (el *EventListener) ValidateEventSource(ctx context.Context) error {
	if el == nil {
		return fmt.Errorf("event listener can't be nil")
	}
	if el.GenericEventSource.URL == "" {
		return fmt.Errorf("server url can't be empty")
	}
	if el.GenericEventSource.Config == "" {
		return fmt.Errorf("config can't be empty")
	}
	return nil
}
