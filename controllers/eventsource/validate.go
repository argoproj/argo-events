package eventsource

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/eventsources"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/pkg/errors"
)

// ValidateEventSource validates if the eventSource is valid
func ValidateEventSource(eventSource *v1alpha1.EventSource) error {
	servers := eventsources.GetEventingServers(eventSource)
	if len(servers) != 1 {
		// We don't allow multiple types of event sources in one EventSource for now.
		eventSource.Status.MarkSourcesNotProvided("InvalidEventSource", "Either no event source or multiple types of event sources found")
		return errors.New("either no event source or multiple types of event sources found")
	}
	ctx := context.Background()
	for _, ss := range servers {
		for _, server := range ss {
			err := server.ValidateEventSource(ctx)
			if err != nil {
				eventSource.Status.MarkSourcesNotProvided("InvalidEventSource", fmt.Sprintf("Invalid spec: %s - %s", server.GetEventSourceName(), server.GetEventName()))
				return err
			}
		}
	}
	eventSource.Status.MarkSourcesProvided()
	return nil
}
