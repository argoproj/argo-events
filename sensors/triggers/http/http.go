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
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

// HTTPTrigger describes the trigger to invoke HTTP request
type Trigger struct {
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *logrus.Logger
}

// NewHTTPTrigger returns a new HTTP trigger
func NewHTTPTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *logrus.Logger) *Trigger {
	return &Trigger{
		Sensor:  sensor,
		Trigger: trigger,
		Logger:  logger,
	}
}

// FetchResource fetches the trigger. As the HTTP trigger simply executes a http request, there
// is no need to fetch any resource from external source
func (t *Trigger) FetchResource() (interface{}, error) {
	return t.Trigger.Template.HTTP, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *Trigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the http trigger resource")
	}
	parameters := t.Trigger.Template.HTTP.ResourceParameters
	if parameters != nil && len(parameters) > 0 {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, t.Trigger.Template.HTTP.ResourceParameters, triggers.ExtractEvents(sensor, parameters))
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
func (t *Trigger) Execute(resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.HTTPTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}

	if trigger.PayloadParameters == nil {
		return nil, errors.New("payload parameters are not specified")
	}

	payload := make(map[string][]byte)

	events := triggers.ExtractEvents(t.Sensor, trigger.PayloadParameters)
	if events == nil {
		return nil, errors.New("payload can't be constructed as there are not events to extract data from")
	}

	for _, parameter := range trigger.PayloadParameters {
		value, err := triggers.ResolveParamValue(parameter.Src, events)
		if err != nil {
			return nil, err
		}
		payload[parameter.Dest] = []byte(value)
	}

	payloadBody, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload")
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

	request, err := http.NewRequest(trigger.Method, trigger.ServerURL, bytes.NewReader(payloadBody))
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
func (t *Trigger) ApplyPolicy(resource interface{}) error {
	if t.Trigger.Policy.HTTP.AllowedStatuses == nil {
		return nil
	}

	response, ok := resource.(*http.Response)
	if !ok {
		return errors.New("failed to interpret the trigger execution response")
	}

	p := policy.NewHTTPTriggerPolicy(response, t.Trigger.Policy.HTTP.AllowedStatuses)

	return p.ApplyPolicy()
}
