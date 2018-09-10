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

package main

import (
	"context"
	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/ghodss/yaml"
	"strconv"
)

const (
	topicKey     = "topic"
	partitionKey = "partition"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// Runs a configuration
func configRunner(config *gateways.ConfigData) error {
	var s *stream.Stream
	err := yaml.Unmarshal([]byte(config.Config), &s)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to parse kafka config")
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Src).Interface("stream", *s).Msg("kafka configuration")
	consumer, err := sarama.NewConsumer([]string{s.URL}, nil)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Str("url", s.URL).Err(err).Msg("failed to connect to cluster")
		return err
	}

	topic := s.Attributes[topicKey]
	pString := s.Attributes[partitionKey]
	pInt, err := strconv.ParseInt(pString, 10, 32)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Str("partition", pString).Err(err).Msg("failed to parse partition key")
		return err
	}
	partition := int32(pInt)

	availablePartitions, err := consumer.Partitions(topic)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Str("topic", topic).Err(err).Msg("unable to get available partitions for kafka topic")
		return err
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Str("partition", pString).Str("topic", topic).Err(err).Msg("partition does not exist for topic")
		return err
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Str("partition", pString).Str("topic", topic).Err(err).Msg("failed to create partition consumer for topic")
		return err
	}

	gatewayConfig.Log.Info().Str("config-name", config.Src).Msg("configuration is running...")
	config.Active = true

	// start listening to messages
kafkaConfigRunner:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("dispatching event to gateway-processor")
			gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Src,
				Payload: msg.Value,
			})
		case err := <-partitionConsumer.Errors():
			gatewayConfig.Log.Error().Str("config-key", config.Src).Str("partition", pString).Str("topic", topic).Err(err).Msg("received an error")
		case <-config.StopCh:
			err = partitionConsumer.Close()
			if err != nil {
				gatewayConfig.Log.Error().Str("config-key", config.Src).Str("partition", pString).Str("topic", topic).Err(err).Msg("failed to close partition")
			}
			break kafkaConfigRunner
		}
	}

	return nil
}

func verifyPartitionAvailable(part int32, partitions []int32) bool {
	for _, p := range partitions {
		if part == p {
			return true
		}
	}
	return false
}

func main() {
	gatewayConfig.WatchGatewayConfigMap(context.Background(), configRunner, core.ConfigDeactivator)
	select {}
}
