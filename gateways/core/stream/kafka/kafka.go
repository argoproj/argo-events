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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"strconv"
	"github.com/argoproj/argo-events/common"
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
func configRunner(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, errMessage, err)

	var s *stream.Stream
	err = yaml.Unmarshal([]byte(config.Data.Config), &s)
	if err != nil {
		errMessage = "failed to parse kafka config"
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("stream", *s).Msg("kafka configuration")
	consumer, err := sarama.NewConsumer([]string{s.URL}, nil)
	if err != nil {
		errMessage = "failed to connect to cluster"
		return err
	}

	topic := s.Attributes[topicKey]
	pString := s.Attributes[partitionKey]
	pInt, err := strconv.ParseInt(pString, 10, 32)
	if err != nil {

		return err
	}
	partition := int32(pInt)

	availablePartitions, err := consumer.Partitions(topic)
	if err != nil {
		errMessage = "unable to get available partitions for kafka topic"
		return err
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		errMessage = "partition does not exist for topic"
		return err
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		errMessage = "failed to create partition consumer for topic"
		return err
	}

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running...")
	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data.Src)
	err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	// start listening to messages
kafkaConfigRunner:
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: msg.Value,
			})
		case err := <-partitionConsumer.Errors():
			gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Str("partition", pString).Str("topic", topic).Err(err).Msg("received an error")
			config.StopCh <- struct{}{}
		case <-config.StopCh:
			err = partitionConsumer.Close()
			if err != nil {
				errMessage = "failed to close partition"
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
	_, err := gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch k8 events for gateway configuration state updates")
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), configRunner, core.ConfigDeactivator)
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}
