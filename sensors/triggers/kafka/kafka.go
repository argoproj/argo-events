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
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hamba/avro"
	"github.com/riferrei/srclient"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
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
	Logger *zap.SugaredLogger
	// Avro schema of message
	schema *srclient.Schema
}

// NewKafkaTrigger returns a new kafka trigger context.
func NewKafkaTrigger(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, kafkaProducers map[string]sarama.AsyncProducer, logger *zap.SugaredLogger) (*KafkaTrigger, error) {
	kafkatrigger := trigger.Template.Kafka
	triggerLogger := logger.With(logging.LabelTriggerType, apicommon.KafkaTrigger)

	producer, ok := kafkaProducers[trigger.Template.Name]
	var schema *srclient.Schema

	if !ok {
		var err error
		config := sarama.NewConfig()

		if kafkatrigger.Version == "" {
			config.Version = sarama.V1_0_0_0
		} else {
			version, err := sarama.ParseKafkaVersion(kafkatrigger.Version)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Kafka version, %w", err)
			}
			config.Version = version
		}

		if kafkatrigger.SASL != nil {
			config.Net.SASL.Enable = true
			config.Net.SASL.Mechanism = sarama.SASLMechanism(kafkatrigger.SASL.GetMechanism())
			if config.Net.SASL.Mechanism == "SCRAM-SHA-512" {
				config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA512New} }
			} else if config.Net.SASL.Mechanism == "SCRAM-SHA-256" {
				config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA256New} }
			}

			user, err := common.GetSecretFromVolume(kafkatrigger.SASL.UserSecret)
			if err != nil {
				return nil, fmt.Errorf("error getting user value from secret, %w", err)
			}
			config.Net.SASL.User = user

			password, err := common.GetSecretFromVolume(kafkatrigger.SASL.PasswordSecret)
			if err != nil {
				return nil, fmt.Errorf("error getting password value from secret, %w", err)
			}
			config.Net.SASL.Password = password
		}

		if kafkatrigger.TLS != nil {
			tlsConfig, err := common.GetTLSConfig(kafkatrigger.TLS)
			if err != nil {
				return nil, fmt.Errorf("failed to get the tls configuration, %w", err)
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

		urls := strings.Split(kafkatrigger.URL, ",")
		producer, err = sarama.NewAsyncProducer(urls, config)
		if err != nil {
			return nil, err
		}

		// must read from the Errors() channel or the async producer will deadlock.
		go func() {
			for err := range producer.Errors() {
				triggerLogger.Errorf("Error happened in kafka producer", err)
			}
		}()

		kafkaProducers[trigger.Template.Name] = producer

		if kafkatrigger.SchemaRegistry != nil {
			var err error
			schema, err = getSchemaFromRegistry(kafkatrigger.SchemaRegistry)
			if err != nil {
				return nil, err
			}
		}
	}

	return &KafkaTrigger{
		Sensor:   sensor,
		Trigger:  trigger,
		Producer: producer,
		Logger:   triggerLogger,
		schema:   schema,
	}, nil
}

// GetTriggerType returns the type of the trigger
func (t *KafkaTrigger) GetTriggerType() apicommon.TriggerType {
	return apicommon.KafkaTrigger
}

// FetchResource fetches the trigger. As the Kafka trigger is simply a Kafka producer, there
// is no need to fetch any resource from external source
func (t *KafkaTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.Kafka, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *KafkaTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.KafkaTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the kafka trigger resource, %w", err)
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var ht *v1alpha1.KafkaTrigger
		if err := json.Unmarshal(updatedResourceBytes, &ht); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated kafka trigger resource after applying resource parameters. %w", err)
		}
		return ht, nil
	}
	return resource, nil
}

// Execute executes the trigger
func (t *KafkaTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.KafkaTrigger)
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

	// Producer with avro schema
	if t.schema != nil {
		payload, err = avroParser(t.schema.Schema(), t.schema.ID(), payload)
		if err != nil {
			return nil, err
		}
	}

	msg := &sarama.ProducerMessage{
		Topic:     trigger.Topic,
		Value:     sarama.ByteEncoder(payload),
		Timestamp: time.Now().UTC(),
	}

	if trigger.PartitioningKey != nil {
		msg.Key = sarama.StringEncoder(*trigger.PartitioningKey)
	}

	t.Producer.Input() <- msg

	t.Logger.Infow("successfully produced a message", zap.Any("topic", trigger.Topic))

	return nil, nil
}

// ApplyPolicy applies policy on the trigger
func (t *KafkaTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	return nil
}

func avroParser(schema string, schemaID int, payload []byte) ([]byte, error) {
	var recordValue []byte
	var payloadNative map[string]interface{}

	schemaAvro, err := avro.Parse(schema)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(payload, &payloadNative)
	if err != nil {
		return nil, err
	}
	avroNative, err := avro.Marshal(schemaAvro, payloadNative)
	if err != nil {
		return nil, err
	}

	schemaIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(schemaIDBytes, uint32(schemaID))
	recordValue = append(recordValue, byte(0))
	recordValue = append(recordValue, schemaIDBytes...)
	recordValue = append(recordValue, avroNative...)

	return recordValue, nil
}

// getSchemaFromRegistry returns a schema from registry.
func getSchemaFromRegistry(sr *apicommon.SchemaRegistryConfig) (*srclient.Schema, error) {
	schemaRegistryClient := srclient.CreateSchemaRegistryClient(sr.URL)
	if sr.Auth.Username != nil && sr.Auth.Password != nil {
		user, _ := common.GetSecretFromVolume(sr.Auth.Username)
		password, _ := common.GetSecretFromVolume(sr.Auth.Password)
		schemaRegistryClient.SetCredentials(user, password)
	}
	schema, err := schemaRegistryClient.GetSchema(int(sr.SchemaID))
	if err != nil {
		return nil, fmt.Errorf("error getting the schema with id '%d' %s", sr.SchemaID, err)
	}
	return schema, nil
}
