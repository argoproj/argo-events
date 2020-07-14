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

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for azure events hub event source
type EventListener struct {
	EventSourceName           string
	EventName                 string
	AzureEventsHubEventSource v1alpha1.AzureEventsHubEventSource
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
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	log.Infoln("started processing the Azure Events Hub event source...")
	defer sources.Recover(el.GetEventName())

	hubEventSource := &el.AzureEventsHubEventSource
	log.Infoln("retrieving the shared access key name...")
	sharedAccessKeyName, ok := common.GetEnvFromSecret(hubEventSource.SharedAccessKeyName)
	if !ok {
		return errors.Errorf("failed to retrieve the shared access key name from secret %s", hubEventSource.SharedAccessKeyName.Name)
	}

	log.Infoln("retrieving the shared access key...")
	sharedAccessKey, ok := common.GetEnvFromSecret(hubEventSource.SharedAccessKey)
	if !ok {
		return errors.Errorf("failed to retrieve the shared access key from secret %s", hubEventSource.SharedAccessKey.Name)
	}

	endpoint := fmt.Sprintf("Endpoint=sb://%s/;SharedAccessKeyName=%s;SharedAccessKey=%s;EntityPath=%s", hubEventSource.FQDN, sharedAccessKeyName, sharedAccessKey, hubEventSource.HubName)

	log.Infoln("connecting to the hub...")
	hub, err := eventhub.NewHubFromConnectionString(endpoint)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the hub %s", hubEventSource.HubName)
	}

	handler := func(c context.Context, event *eventhub.Event) error {
		eventData := &events.AzureEventsHubEventData{
			Id:           event.ID,
			PartitionKey: *event.PartitionKey,
			Body:         event.Data,
		}

		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal the event data for event source %s and message id %s", el.GetEventName(), event.ID)
		}

		err = dispatch(eventBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to dispatch event")
			return err
		}
		return nil
	}

	// listen to each partition of the Event Hub
	log.Infoln("gathering the hub runtime information...")
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get the hub runtime information for %s", el.GetEventName())
	}

	var listenerHandles []*eventhub.ListenerHandle

	log.Infoln("handling the partitions...")
	for _, partitionID := range runtimeInfo.PartitionIDs {
		listenerHandle, err := hub.Receive(ctx, partitionID, handler, eventhub.ReceiveWithLatestOffset())
		if err != nil {
			return errors.Wrapf(err, "failed to receive events from partition %s", partitionID)
		}
		listenerHandles = append(listenerHandles, listenerHandle)
	}

	<-stopCh
	log.Infoln("stopping listener handlers")

	for _, handler := range listenerHandles {
		handler.Close(ctx)
	}

	return nil
}
