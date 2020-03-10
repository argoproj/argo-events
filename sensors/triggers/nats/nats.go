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
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	natslib "github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
	"io/ioutil"
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
