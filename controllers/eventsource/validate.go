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
	servers := eventsources.GetEventingServers(eventSource)
	recreateTypes := make(map[apicommon.EventSourceType]bool)
	for _, esType := range apicommon.RecreateStrategyEventSources {
		recreateTypes[esType] = true
	}
	rollingUpdates, recreates := 0, 0
	for eventType := range servers {
		if _, ok := recreateTypes[eventType]; ok {
			recreates++
		} else {
			rollingUpdates++
		}
	}
	if rollingUpdates > 0 && recreates > 0 {
		// We don't allow this as if we use recreate strategy for the deployment it will have downtime
		eventSource.Status.MarkSourcesNotProvided("InvalidEventSource", "Some types of event sources can not be put in one spec")
		return errors.New("eventsources in the spec require both rolling update and recreate update strategy")
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
