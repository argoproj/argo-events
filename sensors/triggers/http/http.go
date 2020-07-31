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
package http

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
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
	Logger *zap.Logger
}

// NewHTTPTrigger returns a new HTTP trigger
func NewHTTPTrigger(httpClients map[string]*http.Client, sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.Logger) (*HTTPTrigger, error) {
	httptrigger := trigger.Template.HTTP

	client, ok := httpClients[trigger.Template.Name]
	if !ok {
		client = &http.Client{}

		if httptrigger.TLS != nil {
			caCert, err := ioutil.ReadFile(httptrigger.TLS.CACertPath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read ca cert file %s", httptrigger.TLS.CACertPath)
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(caCert)

			clientCert, err := tls.LoadX509KeyPair(httptrigger.TLS.ClientCertPath, httptrigger.TLS.ClientKeyPath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load client cert key pair %s", httptrigger.TLS.CACertPath)
			}
			tlsConfig := tls.Config{
				RootCAs:      pool,
				Certificates: []tls.Certificate{clientCert},
			}
			client.Transport = &http.Transport{
				TLSClientConfig: &tlsConfig,
			}
		}

		timeout := time.Second * 60
		if httptrigger.Timeout > 0 {
			timeout = time.Duration(httptrigger.Timeout) * time.Second
		}
		client.Timeout = timeout

		httpClients[trigger.Template.Name] = client
	}

	return &HTTPTrigger{
		Client:  client,
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger,
	}, nil
}

// FetchResource fetches the trigger. As the HTTP trigger simply executes a http request, there
// is no need to fetch any resource from external source
func (t *HTTPTrigger) FetchResource() (interface{}, error) {
	if t.Trigger.Template.HTTP.Method == "" {
		t.Trigger.Template.HTTP.Method = http.MethodPost
	}
	return t.Trigger.Template.HTTP, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *HTTPTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.HTTPTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the http trigger resource")
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.HTTPTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the updated http trigger resource after applying resource parameters")
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *HTTPTrigger) Execute(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	var payload []byte
	var err error

	trigger, ok := resource.(*v1alpha1.HTTPTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}

	if (trigger.Method == http.MethodPost || trigger.Method == http.MethodPatch || trigger.Method == http.MethodPut) && trigger.Payload == nil {
		t.Logger.Warn("payload parameters are not specified. request payload will be an empty string", zap.Any("url", trigger.URL))
	}

	if trigger.Payload != nil {
		payload, err = triggers.ConstructPayload(events, trigger.Payload)
		if err != nil {
			return nil, err
		}
	}

	request, err := http.NewRequest(trigger.Method, trigger.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to construct request for %s", trigger.URL)
	}

	if trigger.Headers != nil {
		for name, value := range trigger.Headers {
			request.Header[name] = []string{value}
		}
	}

	basicAuth := trigger.BasicAuth

	if basicAuth != nil {
		username := ""
		password := ""

		if basicAuth.Username != nil {
			username, err = common.GetSecretFromVolume(basicAuth.Username)
			if err != nil {
				return nil, errors.Wrap(err, "failed to retrieve the username")
			}
		}

		if basicAuth.Password != nil {
			password, err = common.GetSecretFromVolume(basicAuth.Password)
			if !ok {
				return nil, errors.Wrap(err, "failed to retrieve the password")
			}
		}

		request.SetBasicAuth(username, password)
	}

	t.Logger.Info("making a http request...", zap.Any("url", trigger.URL))

	return t.Client.Do(request)
}

// ApplyPolicy applies policy on the trigger
func (t *HTTPTrigger) ApplyPolicy(resource interface{}) error {
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
