/*
Copyright 2018 The Argoproj Authors.

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
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	eventhub "github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"go.uber.org/zap"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for azure events hub event source
type EventListener struct {
	EventSourceName           string
	EventName                 string
	AzureEventsHubEventSource aev1.AzureEventsHubEventSource
	Metrics                   *metrics.Metrics
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
func (el *EventListener) GetEventSourceType() aev1.EventSourceType {
	return aev1.AzureEventsHub
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	log.Info("started processing the Azure Event Hub event source...")
	defer sources.Recover(el.GetEventName())

	hubEventSource := &el.AzureEventsHubEventSource
	var consumerClient *eventhub.ConsumerClient

	if hubEventSource.SharedAccessKeyName != nil && hubEventSource.SharedAccessKey != nil {
		log.Info("retrieving the shared access key name...")
		sharedAccessKeyName, err := sharedutil.GetSecretFromVolume(hubEventSource.SharedAccessKeyName)
		if err != nil {
			return fmt.Errorf("failed to retrieve the shared access key name from secret %s, %w", hubEventSource.SharedAccessKeyName.Name, err)
		}

		log.Info("retrieving the shared access key...")
		sharedAccessKey, err := sharedutil.GetSecretFromVolume(hubEventSource.SharedAccessKey)
		if err != nil {
			return fmt.Errorf("failed to retrieve the shared access key from secret %s, %w", hubEventSource.SharedAccessKey.Name, err)
		}

		log.Info("connecting to event hub using connection string...")
		endpoint := fmt.Sprintf("Endpoint=sb://%s/;SharedAccessKeyName=%s;SharedAccessKey=%s;EntityPath=%s", hubEventSource.FQDN, sharedAccessKeyName, sharedAccessKey, hubEventSource.HubName)
		consumerClient, err = eventhub.NewConsumerClientFromConnectionString(endpoint, "", eventhub.DefaultConsumerGroup, nil)
		if err != nil {
			return fmt.Errorf("failed to connect to the hub %s, %w", hubEventSource.HubName, err)
		}
	} else {
		credentials, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			log.Errorw("failed to create DefaultAzureCredential", zap.Error(err))
			return err
		}

		log.Info("connecting to event hub using AAD credentials...")
		consumerClient, err = eventhub.NewConsumerClient(hubEventSource.FQDN, hubEventSource.HubName, eventhub.DefaultConsumerGroup, credentials, nil)
		if err != nil {
			return fmt.Errorf("failed to connect to the hub %s, %w", hubEventSource.HubName, err)
		}
	}

	log.Info("retrieving event hub partitions...")
	hubProps, err := consumerClient.GetEventHubProperties(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to get event hub properties: %w", err)
	}

	for _, partitionID := range hubProps.PartitionIDs {
		go func(partitionID string) {
			log.Infow("connecting to partition", "partitionID", partitionID)
			partitionClient, err := consumerClient.NewPartitionClient(partitionID, nil)
			if err != nil {
				log.Errorw("failed to create receiver for partition", "partitionID", partitionID, zap.Error(err))
				return
			}
			defer partitionClient.Close(context.TODO())

			for {
				receiveCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
				receivedEvents, err := partitionClient.ReceiveEvents(receiveCtx, 100, nil)
				cancel()

				if err != nil && !errors.Is(err, context.DeadlineExceeded) {
					log.Errorw("failed to receive events from partition", "partitionID", partitionID, zap.Error(err))
					continue
				}

				for _, event := range receivedEvents {
					log.Infow("received event from partition", "eventID", *event.MessageID, "partitionID", partitionID)
					eventData := &events.AzureEventsHubEventData{
						Id:       *event.MessageID,
						Body:     event.Body,
						Metadata: hubEventSource.Metadata,
					}

					if event.PartitionKey != nil {
						eventData.PartitionKey = *event.PartitionKey
					}

					eventBytes, err := json.Marshal(eventData)
					if err != nil {
						el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
						log.Errorw("failed to marshal the event data", "eventSource", el.GetEventName(), "messageID", *event.MessageID, zap.Error(err))
						continue
					}

					log.Info("dispatching the event to eventbus...")
					if err = dispatch(eventBytes); err != nil {
						el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
						log.Errorw("failed to dispatch Azure EventHub event", zap.Error(err))
						continue
					}
				}
			}
		}(partitionID)
	}

	<-ctx.Done()
	log.Info("stopping listener")

	consumerClient.Close(context.Background())

	return nil
}
