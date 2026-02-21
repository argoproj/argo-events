/*
Copyright 2026 The Argoproj Authors.

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
package gcpcloudfunctions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"google.golang.org/api/idtoken"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/sensors/policy"
	"github.com/argoproj/argo-events/pkg/sensors/triggers"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// GCPCloudFunctionsTrigger describes the trigger to invoke a GCP Cloud Function
type GCPCloudFunctionsTrigger struct {
	// Client is http client.
	Client *http.Client
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *zap.SugaredLogger
}

// NewGCPCloudFunctionsTrigger returns a new GCP Cloud Functions trigger
func NewGCPCloudFunctionsTrigger(gcpClients sharedutil.StringKeyedMap[*http.Client], sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) (*GCPCloudFunctionsTrigger, error) {
	gcpTrigger := trigger.Template.GCPCloudFunctions

	client, ok := gcpClients.Load(trigger.Template.Name)
	if !ok {
		var (
			opts []idtoken.ClientOption
			err  error
		)
		if gcpTrigger.CredentialSecret != nil {
			credJSON, err := sharedutil.GetSecretFromVolume(gcpTrigger.CredentialSecret)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve GCP credential secret, %w", err)
			}
			opts = append(opts, idtoken.WithCredentialsJSON([]byte(credJSON)))
		}

		client, err = idtoken.NewClient(context.Background(), gcpTrigger.URL, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP ID token HTTP client, %w", err)
		}

		timeout := time.Second * 60
		if gcpTrigger.Timeout > 0 {
			timeout = time.Duration(gcpTrigger.Timeout) * time.Second
		}
		client.Timeout = timeout

		gcpClients.Store(trigger.Template.Name, client)
	}

	return &GCPCloudFunctionsTrigger{
		Client:  client,
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger.With(logging.LabelTriggerType, v1alpha1.TriggerTypeGCPCloudFunctions),
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *GCPCloudFunctionsTrigger) GetTriggerType() v1alpha1.TriggerType {
	return v1alpha1.TriggerTypeGCPCloudFunctions
}

// FetchResource fetches the trigger resource
func (t *GCPCloudFunctionsTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	gcpTrigger := t.Trigger.Template.GCPCloudFunctions
	if gcpTrigger.Method == "" {
		gcpTrigger.Method = http.MethodPost
	}
	return gcpTrigger, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *GCPCloudFunctionsTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.GCPCloudFunctionsTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the GCP Cloud Functions trigger resource, %w", err)
	}

	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var gt *v1alpha1.GCPCloudFunctionsTrigger
		if err := json.Unmarshal(updatedResourceBytes, &gt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated GCP Cloud Functions trigger resource after applying resource parameters, %w", err)
		}
		return gt, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *GCPCloudFunctionsTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	var payload []byte
	var err error

	trigger, ok := resource.(*v1alpha1.GCPCloudFunctionsTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the trigger resource")
	}

	if (trigger.Method == http.MethodPost || trigger.Method == http.MethodPatch || trigger.Method == http.MethodPut) && trigger.Payload == nil {
		t.Logger.Warnw("payload parameters are not specified. request payload will be an empty string", zap.Any("url", trigger.URL))
	}

	if trigger.Payload != nil {
		payload, err = triggers.ConstructPayload(events, trigger.Payload)
		if err != nil {
			return nil, err
		}
	}

	request, err := http.NewRequestWithContext(ctx, trigger.Method, trigger.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to construct request for %s, %w", trigger.URL, err)
	}

	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if trigger.Headers != nil {
		for name, value := range trigger.Headers {
			request.Header[name] = []string{value}
		}
	}

	t.Logger.Infow("Invoking GCP Cloud Function...", zap.Any("url", trigger.URL))

	return t.Client.Do(request)
}

// ApplyPolicy applies policy on the trigger execution response
func (t *GCPCloudFunctionsTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	if t.Trigger.Policy == nil || t.Trigger.Policy.Status == nil || t.Trigger.Policy.Status.Allow == nil {
		return nil
	}

	response, ok := resource.(*http.Response)
	if !ok {
		return fmt.Errorf("failed to interpret the trigger execution response")
	}

	p := policy.NewStatusPolicy(response.StatusCode, t.Trigger.Policy.Status.GetAllow())
	return p.ApplyPolicy(ctx)
}
