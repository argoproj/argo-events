/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azureeventshub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for azure events hub event source
type EventListener struct {
	EventSourceName           string
	EventName                 string
	AzureEventsHubEventSource v1alpha1.AzureEventsHubEventSource
	Metrics                   *metrics.Metrics
}

func (el *EventListener) GetEventFilter() *v1alpha1.EventSourceFilter {
	return nil
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.AzureEventsHub
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Options) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the Azure Events Hub event source...")
	defer sources.Recover(el.GetEventName())

	hubEventSource := &el.AzureEventsHubEventSource
	log.Info("retrieving the shared access key name...")
	sharedAccessKeyName, err := common.GetSecretFromVolume(hubEventSource.SharedAccessKeyName)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve the shared access key name from secret %s", hubEventSource.SharedAccessKeyName.Name)
	}

	log.Info("retrieving the shared access key...")
	sharedAccessKey, err := common.GetSecretFromVolume(hubEventSource.SharedAccessKey)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve the shared access key from secret %s", hubEventSource.SharedAccessKey.Name)
	}

	endpoint := fmt.Sprintf("Endpoint=sb://%s/;SharedAccessKeyName=%s;SharedAccessKey=%s;EntityPath=%s", hubEventSource.FQDN, sharedAccessKeyName, sharedAccessKey, hubEventSource.HubName)

	log.Info("connecting to the hub...")
	hub, err := eventhub.NewHubFromConnectionString(endpoint)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the hub %s", hubEventSource.HubName)
	}

	handler := func(c context.Context, event *eventhub.Event) error {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		log.Info("received an event from eventshub...")

		eventData := &events.AzureEventsHubEventData{
			Id:       event.ID,
			Body:     event.Data,
			Metadata: hubEventSource.Metadata,
		}
		if event.PartitionKey != nil {
			eventData.PartitionKey = *event.PartitionKey
		}

		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			return errors.Wrapf(err, "failed to marshal the event data for event source %s and message id %s", el.GetEventName(), event.ID)
		}

		log.Info("dispatching the event to eventbus...")
		if err = dispatch(eventBytes); err != nil {
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			log.Errorw("failed to dispatch Azure EventHub event", zap.Error(err))
			return err
		}
		return nil
	}

	// listen to each partition of the Event Hub
	log.Info("gathering the hub runtime information...")
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get the hub runtime information for %s", el.GetEventName())
	}

	if runtimeInfo == nil {
		return errors.Wrapf(err, "runtime information is not available for %s", el.GetEventName())
	}

	if runtimeInfo.PartitionIDs == nil {
		return errors.Wrapf(err, "no partition ids are available for %s", el.GetEventName())
	}

	log.Info("handling the partitions...")
	for _, partitionID := range runtimeInfo.PartitionIDs {
		if _, err := hub.Receive(ctx, partitionID, handler, eventhub.ReceiveWithLatestOffset()); err != nil {
			return errors.Wrapf(err, "failed to receive events from partition %s", partitionID)
		}
	}

	<-ctx.Done()
	log.Info("stopping listener handlers")

	hub.Close(context.Background())

	return nil
}
