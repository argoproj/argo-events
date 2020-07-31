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
package apache_openwhisk

import (
	"encoding/json"
	"net/http"

	"github.com/apache/openwhisk-client-go/whisk"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
)

// TriggerImpl implements the Trigger interface for OpenWhisk trigger.
type TriggerImpl struct {
	// OpenWhiskClient is OpenWhisk API client
	OpenWhiskClient *whisk.Client
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger definition
	Trigger *v1alpha1.Trigger
	// logger to log stuff
	Logger *zap.Logger
}

// NewTriggerImpl returns a new TriggerImpl
func NewTriggerImpl(openWhiskClients map[string]*whisk.Client, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.Logger) (*TriggerImpl, error) {
	openwhisktrigger := trigger.Template.OpenWhisk

	client, ok := openWhiskClients[trigger.Template.Name]
	if !ok {
		logger.Debug("OpenWhisk trigger value", zap.Any("name", trigger.Template.Name), zap.Any("trigger", *trigger.Template.OpenWhisk))
		logger.Info("instantiating OpenWhisk client", zap.Any("trigger-name", trigger.Template.Name))

		config, err := whisk.GetDefaultConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get default configuration")
		}

		config.Host = openwhisktrigger.Host

		if openwhisktrigger.AuthToken != nil {
			token, err := common.GetSecretFromVolume(openwhisktrigger.AuthToken)
			if err != nil {
				return nil, errors.Wrap(err, "failed to retrieve auth token")
			}
			config.AuthToken = token
		}

		if openwhisktrigger.Namespace != "" {
			config.Namespace = openwhisktrigger.Namespace
		}
		if openwhisktrigger.Version != "" {
			config.Version = openwhisktrigger.Version
		}

		logger.Debug("configuration for OpenWhisk client", zap.Any("config", *config))

		client, err = whisk.NewClient(http.DefaultClient, config)
		if err != nil {
			return nil, errors.Wrap(err, "failed to instantiate OpenWhisk client")
		}

		openWhiskClients[trigger.Template.Name] = client
	}

	return &TriggerImpl{
		OpenWhiskClient: client,
		Sensor:          sensor,
		Trigger:         trigger,
		Logger:          logger,
	}, nil
}

// FetchResource fetches the trigger. As the OpenWhisk trigger simply executes a http request, there
// is no need to fetch any resource from external source
func (t *TriggerImpl) FetchResource() (interface{}, error) {
	return t.Trigger.Template.OpenWhisk, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *TriggerImpl) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.OpenWhiskTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the OpenWhisk trigger resource")
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var openwhisktrigger *v1alpha1.OpenWhiskTrigger
		if err := json.Unmarshal(updatedResourceBytes, &openwhisktrigger); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the updated OpenWhisk trigger resource after applying resource parameters")
		}

		t.Logger.Debug("applied parameters to the OpenWhisk trigger", zap.Any("name", t.Trigger.Template.Name), zap.Any("trigger", *openwhisktrigger))

		return openwhisktrigger, nil
	}

	return resource, nil
}

// Execute executes the trigger
func (t *TriggerImpl) Execute(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	var payload []byte
	var err error

	openwhisktrigger, ok := resource.(*v1alpha1.OpenWhiskTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the OpenWhisk trigger resource")
	}

	if openwhisktrigger.Payload != nil {
		payload, err = triggers.ConstructPayload(events, openwhisktrigger.Payload)
		if err != nil {
			return nil, err
		}

		t.Logger.Debug("payload for the OpenWhisk action invocation", zap.Any("name", t.Trigger.Template.Name), zap.Any("payload", string(payload)))
	}

	response, status, err := t.OpenWhiskClient.Actions.Invoke(openwhisktrigger.ActionName, payload, true, true)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to invoke action %s", openwhisktrigger.ActionName)
	}

	t.Logger.Debug("response for the OpenWhisk action invocation", zap.Any("name", t.Trigger.Template.Name), zap.Any("response", response))

	return status, nil
}

// ApplyPolicy applies policy on the trigger
func (t *TriggerImpl) ApplyPolicy(resource interface{}) error {
	if t.Trigger.Policy == nil || t.Trigger.Policy.Status == nil || t.Trigger.Policy.Status.Allow == nil {
		return nil
	}
	response, ok := resource.(*http.Response)
	if !ok {
		return errors.New("failed to interpret the trigger execution response")
	}

	p := policy.NewStatusPolicy(response.StatusCode, t.Trigger.Policy.Status.GetAllow())

	return p.ApplyPolicy()
}
