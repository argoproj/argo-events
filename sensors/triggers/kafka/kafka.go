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
package kafka

import (
	"encoding/json"
	"time"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/triggers"
)

// KafkaTrigger describes the trigger to place messages on Kafka topic using a producer
type KafkaTrigger struct {
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Kafka async producer
	Producer sarama.AsyncProducer
	// Logger to log stuff
	Logger *zap.Logger
}

// NewKafkaTrigger returns a new kafka trigger context.
func NewKafkaTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, kafkaProducers map[string]sarama.AsyncProducer, logger *zap.Logger) (*KafkaTrigger, error) {
	kafkatrigger := trigger.Template.Kafka

	producer, ok := kafkaProducers[trigger.Template.Name]
	if !ok {
		var err error
		config := sarama.NewConfig()

		if kafkatrigger.TLS != nil {
			tlsConfig, err := common.GetTLSConfig(kafkatrigger.TLS)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get the tls configuration")
			}
			tlsConfig.InsecureSkipVerify = true
			config.Net.TLS.Config = tlsConfig
			config.Net.TLS.Enable = true
		}

		if kafkatrigger.Compress {
			config.Producer.Compression = sarama.CompressionSnappy
		}

		ff := 500
		if kafkatrigger.FlushFrequency != 0 {
			ff = int(kafkatrigger.FlushFrequency)
		}
		config.Producer.Flush.Frequency = time.Duration(ff)

		ra := sarama.WaitForAll
		if kafkatrigger.RequiredAcks != 0 {
			ra = sarama.RequiredAcks(kafkatrigger.RequiredAcks)
		}
		config.Producer.RequiredAcks = ra

		producer, err = sarama.NewAsyncProducer([]string{kafkatrigger.URL}, config)
		if err != nil {
			return nil, err
		}

		kafkaProducers[trigger.Template.Name] = producer
	}

	return &KafkaTrigger{
		Sensor:   sensor,
		Trigger:  trigger,
		Producer: producer,
		Logger:   logger,
	}, nil
}

// FetchResource fetches the trigger. As the Kafka trigger is simply a Kafka producer, there
// is no need to fetch any resource from external source
func (t *KafkaTrigger) FetchResource() (interface{}, error) {
	return t.Trigger.Template.Kafka, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *KafkaTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.KafkaTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal the kafka trigger resource")
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.KafkaTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal the updated kafka trigger resource after applying resource parameters")
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *KafkaTrigger) Execute(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.KafkaTrigger)
	if !ok {
		return nil, errors.New("failed to interpret the trigger resource")
	}

	if trigger.Payload == nil {
		return nil, errors.New("payload parameters are not specified")
	}

	payload, err := triggers.ConstructPayload(events, trigger.Payload)
	if err != nil {
		return nil, err
	}

	pk := trigger.PartitioningKey
	if pk == "" {
		pk = trigger.URL
	}

	t.Producer.Input() <- &sarama.ProducerMessage{
		Topic:     trigger.Topic,
		Key:       sarama.StringEncoder(pk),
		Value:     sarama.ByteEncoder(payload),
		Partition: trigger.Partition,
		Timestamp: time.Now().UTC(),
	}

	t.Logger.Info("successfully produced a message", zap.Any("topic", trigger.Topic), zap.Any("partition", trigger.Partition))

	return nil, nil
}

// ApplyPolicy applies policy on the trigger
func (t *KafkaTrigger) ApplyPolicy(resource interface{}) error {
	return nil
}
