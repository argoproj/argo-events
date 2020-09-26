package generic

import (
	"context"
	"fmt"
)

func (el *EventListener) ValidateEventSource(ctx context.Context) error {
	if &el.GenericEventSource == nil {
		return fmt.Errorf("event source can't be empty")
	}
	if el.GenericEventSource.URL == "" {
		return fmt.Errorf("server url can't be empty")
	}
	if el.GenericEventSource.Config == "" {
		return fmt.Errorf("config can't be empty")
	}
	return nil
}
