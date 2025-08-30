/*
Copyright 2020 The Argoproj Authors.

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
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/sensors/policy"
	"github.com/argoproj/argo-events/pkg/sensors/triggers"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// HTTPTrigger describes the trigger to invoke HTTP request
type HTTPTrigger struct {
	// Client is http client.
	Client *http.Client
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *zap.SugaredLogger
}

// NewHTTPTrigger returns a new HTTP trigger
func NewHTTPTrigger(httpClients sharedutil.StringKeyedMap[*http.Client], sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) (*HTTPTrigger, error) {
	httptrigger := trigger.Template.HTTP

	client, ok := httpClients.Load(trigger.Template.Name)
	if !ok {
		client = &http.Client{}

		if httptrigger.TLS != nil {
			tlsConfig, err := sharedutil.GetTLSConfig(httptrigger.TLS)
			if err != nil {
				return nil, fmt.Errorf("failed to get the tls configuration, %w", err)
			}
			client.Transport = &http.Transport{
				TLSClientConfig: tlsConfig,
			}
		}

		timeout := time.Second * 60
		if httptrigger.Timeout > 0 {
			timeout = time.Duration(httptrigger.Timeout) * time.Second
		}
		client.Timeout = timeout

		httpClients.Store(trigger.Template.Name, client)
	}

	return &HTTPTrigger{
		Client:  client,
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger.With(logging.LabelTriggerType, v1alpha1.TriggerTypeHTTP),
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *HTTPTrigger) GetTriggerType() v1alpha1.TriggerType {
	return v1alpha1.TriggerTypeHTTP
}

// FetchResource fetches the trigger. As the HTTP trigger simply executes a http request, there
// is no need to fetch any resource from external source
func (t *HTTPTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	if t.Trigger.Template.HTTP.Method == "" {
		t.Trigger.Template.HTTP.Method = http.MethodPost
	}
	return t.Trigger.Template.HTTP, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *HTTPTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.HTTPTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the http trigger resource, %w", err)
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.HTTPTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated http trigger resource after applying resource parameters, %w", err)
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *HTTPTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	var payload []byte
	var err error

	trigger, ok := resource.(*v1alpha1.HTTPTrigger)
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

	request, err := http.NewRequest(trigger.Method, trigger.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to construct request for %s, %w", trigger.URL, err)
	}

	if trigger.Headers != nil {
		for name, value := range trigger.Headers {
			request.Header[name] = []string{value}
		}
	}

	if trigger.DynamicHeaders != nil {
		for _, dynamic := range trigger.DynamicHeaders {
			value, _, err := triggers.ResolveParamValue(dynamic.Src, events)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve the value for dynamicHeader %s, %w", dynamic.Dest, err)
			}
			// N.B - Removed 'if' clause - flagged by linter as not required,
			request.Header[dynamic.Dest] = []string{*value}
		}
	}

	if trigger.SecureHeaders != nil {
		for _, secure := range trigger.SecureHeaders {
			var value string
			var err error
			if secure.ValueFrom.SecretKeyRef != nil {
				value, err = sharedutil.GetSecretFromVolume(secure.ValueFrom.SecretKeyRef)
			} else {
				value, err = sharedutil.GetConfigMapFromVolume(secure.ValueFrom.ConfigMapKeyRef)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve the value for secureHeader, %w", err)
			}
			request.Header[secure.Name] = []string{value}
		}
	}

	basicAuth := trigger.BasicAuth

	if basicAuth != nil {
		username := ""
		password := ""

		if basicAuth.Username != nil {
			username, err = sharedutil.GetSecretFromVolume(basicAuth.Username)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve the username, %w", err)
			}
		}

		if basicAuth.Password != nil {
			password, err = sharedutil.GetSecretFromVolume(basicAuth.Password)
			if !ok {
				return nil, fmt.Errorf("failed to retrieve the password, %w", err)
			}
		}

		request.SetBasicAuth(username, password)
	}

	t.Logger.Infow("Making a http request...", zap.Any("url", trigger.URL))

	return t.Client.Do(request)
}

// ApplyPolicy applies policy on the trigger
func (t *HTTPTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
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
