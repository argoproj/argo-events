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

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// HTTPTrigger describes the trigger to invoke HTTP request
type HTTPTrigger struct {
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *logrus.Logger
}

// NewHTTPTrigger returns a new HTTP trigger
func NewHTTPTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger) *HTTPTrigger {
	return &HTTPTrigger{
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger,
	}
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
func (t *HTTPTrigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
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
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, triggers.ExtractEvents(sensor, parameters))
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
func (t *HTTPTrigger) Execute(resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.HTTPTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}

	if trigger.Payload == nil {
		return nil, errors.New("payload parameters are not specified")
	}

	payload, err := triggers.ConstructPayload(t.Sensor, trigger.Payload)
	if err != nil {
		return nil, err
	}

	client := http.Client{}

	if trigger.TLS != nil {
		caCert, err := ioutil.ReadFile(trigger.TLS.CACertPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read ca cert file %s", trigger.TLS.CACertPath)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)

		clientCert, err := tls.LoadX509KeyPair(trigger.TLS.ClientCertPath, trigger.TLS.ClientKeyPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load client cert key pair %s", trigger.TLS.CACertPath)
		}
		tlsConfig := tls.Config{
			RootCAs:      pool,
			Certificates: []tls.Certificate{clientCert},
		}
		transport := http.Transport{
			TLSClientConfig: &tlsConfig,
		}
		client = http.Client{
			Transport: &transport,
		}
	}

	request, err := http.NewRequest(trigger.Method, trigger.ServerURL, bytes.NewReader(payload))
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct the http request")
	}

	timeout := time.Second * 10
	if trigger.Timeout > 0 {
		timeout = time.Duration(trigger.Timeout) * time.Second
	}
	client.Timeout = timeout

	return client.Do(request)
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

	p := policy.NewStatusPolicy(response.StatusCode, t.Trigger.Policy.Status.Allow)

	return p.ApplyPolicy()
}
