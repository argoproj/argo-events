/*
Copyright 2018 The Argoproj Authors.

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
	"context"
	"encoding/json"
	"fmt"

	natslib "github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/sensors/triggers"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
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
	Logger *zap.SugaredLogger
}

// NewNATSTrigger returns new nats trigger.
func NewNATSTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, natsConnections sharedutil.StringKeyedMap[*natslib.Conn], logger *zap.SugaredLogger) (*NATSTrigger, error) {
	natstrigger := trigger.Template.NATS

	conn, ok := natsConnections.Load(trigger.Template.Name)
	if !ok {
		var err error

		var opt []natslib.Option

		if natstrigger.TLS != nil {
			tlsConfig, err := sharedutil.GetTLSConfig(natstrigger.TLS)
			if err != nil {
				return nil, fmt.Errorf("failed to get the tls configuration, %w", err)
			}
			opt = append(opt, natslib.Secure(tlsConfig))
		}

		if natstrigger.Auth != nil {
			switch {
			case natstrigger.Auth.Basic != nil:
				username, err := sharedutil.GetSecretFromVolume(natstrigger.Auth.Basic.Username)
				if err != nil {
					return nil, err
				}
				password, err := sharedutil.GetSecretFromVolume(natstrigger.Auth.Basic.Password)
				if err != nil {
					return nil, err
				}
				opt = append(opt, natslib.UserInfo(username, password))
			case natstrigger.Auth.Token != nil:
				token, err := sharedutil.GetSecretFromVolume(natstrigger.Auth.Token)
				if err != nil {
					return nil, err
				}
				opt = append(opt, natslib.Token(token))
			case natstrigger.Auth.NKey != nil:
				nkeyFile, err := sharedutil.GetSecretVolumePath(natstrigger.Auth.NKey)
				if err != nil {
					return nil, err
				}
				o, err := natslib.NkeyOptionFromSeed(nkeyFile)
				if err != nil {
					return nil, fmt.Errorf("failed to get NKey, %w", err)
				}
				opt = append(opt, o)
			case natstrigger.Auth.Credential != nil:
				cFile, err := sharedutil.GetSecretVolumePath(natstrigger.Auth.Credential)
				if err != nil {
					return nil, err
				}
				opt = append(opt, natslib.UserCredentials(cFile))
			}
		}

		conn, err = natslib.Connect(natstrigger.URL, opt...)
		if err != nil {
			return nil, err
		}

		natsConnections.Store(trigger.Template.Name, conn)
	}

	return &NATSTrigger{
		Sensor:  sensor,
		Trigger: trigger,
		Conn:    conn,
		Logger:  logger.With(logging.LabelTriggerType, v1alpha1.TriggerTypeNATS),
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *NATSTrigger) GetTriggerType() v1alpha1.TriggerType {
	return v1alpha1.TriggerTypeNATS
}

// FetchResource fetches the trigger. As the NATS trigger is simply a NATS client, there
// is no need to fetch any resource from external source
func (t *NATSTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.NATS, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *NATSTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.NATSTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the nats trigger resource, %w", err)
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.NATSTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated nats trigger resource after applying resource parameters, %w", err)
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *NATSTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.NATSTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the trigger resource")
	}

	if trigger.Payload == nil {
		return nil, fmt.Errorf("payload parameters are not specified")
	}

	payload, err := triggers.ConstructPayload(events, trigger.Payload)
	if err != nil {
		return nil, err
	}

	if err := t.Conn.Publish(t.Trigger.Template.NATS.Subject, payload); err != nil {
		return nil, err
	}

	return nil, nil
}

// ApplyPolicy applies policy on the trigger
func (t *NATSTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	return nil
}
