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
	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"strconv"
	"fmt"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// kafkaConfigExecutor implements ConfigExecutor interface
type kafkaConfigExecutor struct{}

// Runs a configuration
func (kce *kafkaConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	kafkaConfig := config.Data.Config.(*kafka)
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *kafkaConfig).Msg("kafka configuration")

	consumer, err := sarama.NewConsumer([]string{kafkaConfig.URL}, nil)
	if err != nil {
		errMessage = "failed to connect to cluster"
		return err
	}

	pInt, err := strconv.ParseInt(kafkaConfig.Partition, 10, 32)
	if err != nil {

		return err
	}
	partition := int32(pInt)

	availablePartitions, err := consumer.Partitions(kafkaConfig.Topic)
	if err != nil {
		errMessage = "unable to get available partitions for kafka topic"
		return err
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		errMessage = "partition does not exist for topic"
		return err
	}

	partitionConsumer, err := consumer.ConsumePartition(kafkaConfig.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		errMessage = "failed to create partition consumer for topic"
		return err
	}

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running...")
	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
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
			gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Str("partition", kafkaConfig.Partition).Str("topic", kafkaConfig.Topic).Err(err).Msg("received an error")
			config.StopCh <- struct{}{}
		case <-config.StopCh:
			err = partitionConsumer.Close()
			if err != nil {
				errMessage = "failed to close partition"
			}
			break kafkaConfigRunner
		}
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// StopConfiguration stops a configuration
func (kce *kafkaConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

// Validate validates the gateway configuration
func (kce *kafkaConfigExecutor) Validate(config *gateways.ConfigContext) error {
	kafkaConfig, ok := config.Data.Config.(*kafka)
	if !ok {
		return gateways.ErrConfigParseFailed
	}
	if kafkaConfig.URL == "" {
		return fmt.Errorf("%+v, url must be specified", gateways.ErrInvalidConfig)
	}
	if kafkaConfig.Topic == "" {
		return fmt.Errorf("%+v, topic must be specified", gateways.ErrInvalidConfig)
	}
	if kafkaConfig.Partition == "" {
		return fmt.Errorf("%+v, partition must be specified", gateways.ErrInvalidConfig)
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
	gatewayConfig.StartGateway(&kafkaConfigExecutor{})
}
