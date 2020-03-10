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
package nats

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
	natslib "github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NATSTrigger holds the context of the NATS trigger.
type NATSTrigger struct {
	// Sensor object.
	Sensor *v1alpha1.Sensor
	// Trigger reference.
	Trigger *v1alpha1.Trigger
	// Conn refers to the NATS client connection.
	Conn *natslib.Conn
	// Logger to log stuff.
	Logger *logrus.Logger
}

// NewNATSTrigger returns new nats trigger.
func NewNATSTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, natsConnections map[string]*natslib.Conn, logger *logrus.Logger) (*NATSTrigger, error) {
	natstrigger := trigger.Template.NATS

	conn, ok := natsConnections[trigger.Template.Name]
	if !ok {
		var err error
		opts := natslib.GetDefaultOptions()
		opts.Url = natstrigger.URL

		if natstrigger.TLS != nil {
			if natstrigger.TLS.ClientCertPath != "" && natstrigger.TLS.ClientKeyPath != "" && natstrigger.TLS.CACertPath != "" {
				cert, err := tls.LoadX509KeyPair(natstrigger.TLS.ClientCertPath, natstrigger.TLS.ClientKeyPath)
				if err != nil {
					return nil, err
				}

				caCert, err := ioutil.ReadFile(natstrigger.TLS.CACertPath)
				if err != nil {
					return nil, err
				}

				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)

				t := &tls.Config{
					Certificates:       []tls.Certificate{cert},
					RootCAs:            caCertPool,
					InsecureSkipVerify: true,
				}
				opts.Secure = true
				opts.TLSConfig = t
			}
		}

		conn, err = opts.Connect()
		if err != nil {
			return nil, err
		}

		natsConnections[trigger.Template.Name] = conn
	}

	return &NATSTrigger{
		Sensor:  sensor,
		Trigger: trigger,
		Conn:    conn,
		Logger:  logger,
	}, nil
}

// FetchResource fetches the trigger. As the NATS trigger is simply a NATS client, there
// is no need to fetch any resource from external source
func (t *NATSTrigger) FetchResource() (interface{}, error) {
	return t.Trigger.Template.NATS, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *NATSTrigger) ApplyResourceParameters(sensor *v1alpha1.Sensor, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.NATSTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the nats trigger resource")
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, triggers.ExtractEvents(sensor, parameters))
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.NATSTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the updated nats trigger resource after applying resource parameters")
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *NATSTrigger) Execute(resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.NATSTrigger)
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

	if err := t.Conn.Publish(t.Trigger.Template.NATS.Subject, payload); err != nil {
		return nil, err
	}

	return nil, nil
}

// ApplyPolicy applies policy on the trigger
func (t *NATSTrigger) ApplyPolicy(resource interface{}) error {
	return nil
}
