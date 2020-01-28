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

// Data refers to the event data
type Data struct {
	// Id of the message
	Id string `json:"id"`
	// PartitionKey
	PartitionKey string `json:"partitionKey"`
	// Message body
	Body []byte `json:"body"`
}

func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer server.Recover(eventSource.Name)

	log := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("started processing the event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)
	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	var hubEventSource *v1alpha1.AzureEventsHubEventSource
	if err := yaml.Unmarshal(eventSource.Value, &hubEventSource); err != nil {
		errorCh <- err
		return
	}

	sharedAccessKeyName, err := common.GetSecrets(listener.K8sClient, hubEventSource.Namespace, hubEventSource.SharedAccessKeyName)
	if err != nil {
		errorCh <- errors.Wrapf(err, "failed to retrieve the shared access key name from secret %s", hubEventSource.SharedAccessKeyName.Name)
		return
	}

	sharedAccessKey, err := common.GetSecrets(listener.K8sClient, hubEventSource.Namespace, hubEventSource.SharedAccessKey)
	if err != nil {
		errorCh <- errors.Wrapf(err, "failed to retrieve the shared access key from secret %s", hubEventSource.SharedAccessKey.Name)
		return
	}

	endpoint := fmt.Sprintf("Endpoint=sb://%s/;SharedAccessKeyName=%s;SharedAccessKey=%s;EntityPath=%s", hubEventSource.FQDN, sharedAccessKeyName, sharedAccessKey, hubEventSource.HubName)

	hub, err := eventhub.NewHubFromConnectionString(endpoint)
	if err != nil {
		errorCh <- errors.Wrapf(err, "failed to connect to the hub %s", hubEventSource.HubName)
		return
	}

	handler := func(c context.Context, event *eventhub.Event) error {
		eventData := &Data{
			Id:           event.ID,
			PartitionKey: *event.PartitionKey,
			Body:         event.Data,
		}

		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal the event data for event source %s and message id %s", eventSource.Name, event.ID)
		}

		dataCh <- eventBytes
		return nil
	}

	ctx := context.Background()

	// listen to each partition of the Event Hub
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	var listenerHandles []*eventhub.ListenerHandle

	for _, partitionID := range runtimeInfo.PartitionIDs {
		listenerHandle, err := hub.Receive(ctx, partitionID, handler, eventhub.ReceiveWithLatestOffset())
		if err != nil {
			errorCh <- errors.Wrapf(err, "failed to receive events from partition %s", partitionID)
			continue
		}
		listenerHandles = append(listenerHandles, listenerHandle)
	}

	<-doneCh

	listener.Logger.Infoln("stopping listener handlers")

	for _, handler := range listenerHandles {
		handler.Close(ctx)
	}
}
