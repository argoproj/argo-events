package eventsource

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/eventsources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// ValidateEventSource validates if the eventSource is valid
func ValidateEventSource(eventSource *v1alpha1.EventSource) error {
	recreateTypes := make(map[apicommon.EventSourceType]bool)
	for _, esType := range apicommon.RecreateStrategyEventSources {
		recreateTypes[esType] = true
	}

	servers := eventsources.GetEventingServers(eventSource)

	eventNames := make(map[string]bool)
	rollingUpdates, recreates := 0, 0
	ctx := context.Background()
	for eventType, ss := range servers {
		if _, ok := recreateTypes[eventType]; ok {
			recreates++
		} else {
			rollingUpdates++
		}

		for _, server := range ss {
			eName := server.GetEventName()
			if _, ok := eventNames[eName]; !ok {
				eventNames[eName] = true
			} else {
				// Duplicated event name not allowed in one EventSource, even they are in different EventSourceType.
				eventSource.Status.MarkSourcesNotProvided("InvalidEventSource", fmt.Sprintf("more than one \"%s\" found", eName))
				return errors.Errorf("more than one \"%s\" found in the spec", eName)
			}

			err := server.ValidateEventSource(ctx)
			if err != nil {
				eventSource.Status.MarkSourcesNotProvided("InvalidEventSource", fmt.Sprintf("Invalid spec: %s - %s", server.GetEventSourceName(), server.GetEventName()))
				return err
			}
		}
	}

	if rollingUpdates > 0 && recreates > 0 {
		// We don't allow this as if we use recreate strategy for the deployment it will have downtime
		eventSource.Status.MarkSourcesNotProvided("InvalidEventSource", "Some types of event sources can not be put in one spec")
		return errors.New("event sources with rolling update and recreate update strategy can not put together")
	}

	eventSource.Status.MarkSourcesProvided()
	return nil
}
