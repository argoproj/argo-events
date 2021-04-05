/*
Copyright 2020 BlackRock, Inc.

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
package azure_event_hubs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
)

// AzureEventHubsTrigger describes the trigger to send messages to an Event Hub
type AzureEventHubsTrigger struct {
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Hub refers to Azure Event Hub struct
	Hub *eventhub.Hub
	// Logger to log stuff
	Logger *zap.SugaredLogger
}

// NewAzureEventHubsTrigger returns a new azure event hubs context.
func NewAzureEventHubsTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, azureEventHubsClient map[string]*eventhub.Hub, logger *zap.SugaredLogger) (*AzureEventHubsTrigger, error) {
	azureEventHubsTrigger := trigger.Template.AzureEventHubs

	hub, ok := azureEventHubsClient[trigger.Template.Name]

	if !ok {
		// form event hubs connection string in the ff format:
		// Endpoint=sb://namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=superSecret1234=;EntityPath=hubName
		fqdn := azureEventHubsTrigger.FQDN
		hubName := azureEventHubsTrigger.HubName

		sharedAccessKeyName, err := common.GetSecretFromVolume(azureEventHubsTrigger.SharedAccessKeyName)
		if err != nil {
			return nil, err
		}
		sharedAccessKey, err := common.GetSecretFromVolume(azureEventHubsTrigger.SharedAccessKey)
		if err != nil {
			return nil, err
		}

		logger.Debug("generating connection string")
		connStr := fmt.Sprintf("Endpoint=sb://%s/;SharedAccessKeyName=%s;SharedAccessKey=%s;EntityPath=%s", fqdn, sharedAccessKeyName, sharedAccessKey, hubName)
		logger.Debug("connection string: ", connStr)

		hub, err = eventhub.NewHubFromConnectionString(connStr)
		if err != nil {
			return nil, err
		}

		azureEventHubsClient[trigger.Template.Name] = hub
	}

	return &AzureEventHubsTrigger{
		Sensor:  sensor,
		Trigger: trigger,
		Hub:     hub,
		Logger:  logger.With(logging.LabelTriggerType, apicommon.AzureEventHubsTrigger),
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *AzureEventHubsTrigger) GetTriggerType() apicommon.TriggerType {
	return apicommon.AzureEventHubsTrigger
}

// FetchResource fetches the trigger. As the Azure Event Hubs trigger is simply a Hub client, there
// is no need to fetch any resource from external source
func (t *AzureEventHubsTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.AzureEventHubs, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *AzureEventHubsTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.AzureEventHubsTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the azure event hubs trigger resource")
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.AzureEventHubsTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the updated azure event hubs trigger resource after applying resource parameters")
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *AzureEventHubsTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.AzureEventHubsTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}

	if trigger.Payload == nil {
		return nil, errors.New("payload parameters are not specified")
	}

	payload, err := triggers.ConstructPayload(events, trigger.Payload)
	if err != nil {
		return nil, err
	}

	if err := t.Hub.Send(ctx, eventhub.NewEvent(payload)); err != nil {
		return nil, err
	}
	t.Logger.Info("successfully sent an event to Azure Event Hubs")

	return nil, nil
}

// ApplyPolicy applies policy on the trigger
func (t *AzureEventHubsTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	return nil
}
