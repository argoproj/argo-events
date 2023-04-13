/*
Copyright 2021 BlackRock, Inc.

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
package pulsar

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
)

// PulsarTrigger describes the trigger to place messages on Pulsar topic using a producer
type PulsarTrigger struct {
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Pulsar async producer
	Producer pulsar.Producer
	// Logger to log stuff
	Logger *zap.SugaredLogger
}

// NewPulsarTrigger returns a new Pulsar trigger context.
func NewPulsarTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, pulsarProducers map[string]pulsar.Producer, logger *zap.SugaredLogger) (*PulsarTrigger, error) {
	pulsarTrigger := trigger.Template.Pulsar

	producer, ok := pulsarProducers[trigger.Template.Name]
	if !ok {
		var err error
		tlsTrustCertsFilePath := ""
		if pulsarTrigger.TLSTrustCertsSecret != nil {
			tlsTrustCertsFilePath, err = common.GetSecretVolumePath(pulsarTrigger.TLSTrustCertsSecret)
			if err != nil {
				logger.Errorw("failed to get TLSTrustCertsFilePath from the volume", zap.Error(err))
				return nil, err
			}
		}
		clientOpt := pulsar.ClientOptions{
			URL:                        pulsarTrigger.URL,
			TLSTrustCertsFilePath:      tlsTrustCertsFilePath,
			TLSAllowInsecureConnection: pulsarTrigger.TLSAllowInsecureConnection,
			TLSValidateHostname:        pulsarTrigger.TLSValidateHostname,
		}

		if pulsarTrigger.AuthTokenSecret != nil {
			token, err := common.GetSecretFromVolume(pulsarTrigger.AuthTokenSecret)
			if err != nil {
				logger.Errorw("failed to get AuthTokenSecret from the volume", zap.Error(err))
				return nil, err
			}
			clientOpt.Authentication = pulsar.NewAuthenticationToken(token)
		}

		if len(pulsarTrigger.AuthAthenzParams) > 0 && pulsarTrigger.AuthAthenzSecret != nil {
			logger.Info("setting athenz auth option...")
			authAthenzFilePath, err := common.GetSecretVolumePath(pulsarTrigger.AuthAthenzSecret)
			if err != nil {
				logger.Errorw("failed to get authAthenzSecret from the volume", zap.Error(err))
				return nil, err
			}
			pulsarTrigger.AuthAthenzParams["privateKey"] = "file://" + authAthenzFilePath
			clientOpt.Authentication = pulsar.NewAuthenticationAthenz(pulsarTrigger.AuthAthenzParams)
		}

		if pulsarTrigger.TLS != nil {
			logger.Info("setting tls auth option...")
			var clientCertPath, clientKeyPath string
			switch {
			case pulsarTrigger.TLS.ClientCertSecret != nil && pulsarTrigger.TLS.ClientKeySecret != nil:
				clientCertPath, err = common.GetSecretVolumePath(pulsarTrigger.TLS.ClientCertSecret)
				if err != nil {
					logger.Errorw("failed to get ClientCertPath from the volume", zap.Error(err))
					return nil, err
				}
				clientKeyPath, err = common.GetSecretVolumePath(pulsarTrigger.TLS.ClientKeySecret)
				if err != nil {
					logger.Errorw("failed to get ClientKeyPath from the volume", zap.Error(err))
					return nil, err
				}
			default:
				return nil, fmt.Errorf("invalid TLS config")
			}
			clientOpt.Authentication = pulsar.NewAuthenticationTLS(clientCertPath, clientKeyPath)
		}

		var client pulsar.Client

		if err := common.DoWithRetry(pulsarTrigger.ConnectionBackoff, func() error {
			var err error
			if client, err = pulsar.NewClient(clientOpt); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("failed to connect to %s for sensor %s, %w", pulsarTrigger.URL, trigger.Template.Name, err)
		}

		producer, err = client.CreateProducer(pulsar.ProducerOptions{
			Topic: pulsarTrigger.Topic,
		})
		if err != nil {
			return nil, err
		}

		pulsarProducers[trigger.Template.Name] = producer
	}

	return &PulsarTrigger{
		Sensor:   sensor,
		Trigger:  trigger,
		Producer: producer,
		Logger:   logger.With(logging.LabelTriggerType, apicommon.PulsarTrigger),
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *PulsarTrigger) GetTriggerType() apicommon.TriggerType {
	return apicommon.PulsarTrigger
}

// FetchResource fetches the trigger. As the Pulsar trigger is simply a Pulsar producer, there
// is no need to fetch any resource from external source
func (t *PulsarTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.Pulsar, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *PulsarTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.PulsarTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the pulsar trigger resource, %w", err)
	}

	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.PulsarTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated pulsar trigger resource after applying resource parameters, %w", err)
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *PulsarTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.PulsarTrigger)
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

	_, err = t.Producer.Send(ctx, &pulsar.ProducerMessage{
		Payload: payload,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send message to pulsar, %w", err)
	}

	t.Logger.Infow("successfully produced a message", zap.Any("topic", trigger.Topic))

	return nil, nil
}

// ApplyPolicy applies policy on the trigger
func (t *PulsarTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	return nil
}
