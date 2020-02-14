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

package azure_events_hub

import (
	"context"
	"encoding/json"
	"fmt"

	eventhub "github.com/Azure/azure-event-hubs-go"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	common2 "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
	// k8sClient is kubernetes client
	K8sClient kubernetes.Interface
}

func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")
	channels := server.NewChannels()

	go server.HandleEventsFromEventSource(eventSource.Name, eventStream, channels, listener.Logger)

	defer func() {
		channels.Stop <- struct{}{}
	}()

	if err := listener.listenEvents(eventSource, channels); err != nil {
		listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to listen to events")
		return err
	}

	return nil
}

func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var hubEventSource *v1alpha1.AzureEventsHubEventSource
	if err := yaml.Unmarshal(eventSource.Value, &hubEventSource); err != nil {
		return errors.Wrapf(err, "failed to parsed the event source %s", eventSource.Name)
	}

	logger.Infoln("retrieving the shared access key name...")
	sharedAccessKeyName, err := common.GetSecrets(listener.K8sClient, hubEventSource.Namespace, hubEventSource.SharedAccessKeyName)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve the shared access key name from secret %s", hubEventSource.SharedAccessKeyName.Name)
	}

	logger.Infoln("retrieving the shared access key...")
	sharedAccessKey, err := common.GetSecrets(listener.K8sClient, hubEventSource.Namespace, hubEventSource.SharedAccessKey)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve the shared access key from secret %s", hubEventSource.SharedAccessKey.Name)
	}

	endpoint := fmt.Sprintf("Endpoint=sb://%s/;SharedAccessKeyName=%s;SharedAccessKey=%s;EntityPath=%s", hubEventSource.FQDN, sharedAccessKeyName, sharedAccessKey, hubEventSource.HubName)

	logger.Infoln("connecting to the hub...")
	hub, err := eventhub.NewHubFromConnectionString(endpoint)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to the hub %s", hubEventSource.HubName)
	}

	handler := func(c context.Context, event *eventhub.Event) error {
		eventData := &common2.AzureEventsHubEventData{
			Id:           event.ID,
			PartitionKey: *event.PartitionKey,
			Body:         event.Data,
		}

		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal the event data for event source %s and message id %s", eventSource.Name, event.ID)
		}

		channels.Data <- eventBytes
		return nil
	}

	ctx := context.Background()

	// listen to each partition of the Event Hub
	logger.Infoln("gathering the hub runtime information...")
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get the hub runtime information for %s", eventSource.Name)
	}

	var listenerHandles []*eventhub.ListenerHandle

	logger.Infoln("handling the partitions...")
	for _, partitionID := range runtimeInfo.PartitionIDs {
		listenerHandle, err := hub.Receive(ctx, partitionID, handler, eventhub.ReceiveWithLatestOffset())
		if err != nil {
			return errors.Wrapf(err, "failed to receive events from partition %s", partitionID)
		}
		listenerHandles = append(listenerHandles, listenerHandle)
	}

	<-channels.Done

	listener.Logger.Infoln("stopping listener handlers")

	for _, handler := range listenerHandles {
		handler.Close(ctx)
	}

	return nil
}
