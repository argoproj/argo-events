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
	"fmt"
	"strconv"

	"bytes"
	"context"
	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/ghodss/yaml"
	hs "github.com/mitchellh/hashstructure"
	zlog "github.com/rs/zerolog"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
)

const (
	topicKey     = "topic"
	partitionKey = "partition"
)

type kafka struct {
	gatewayConfig          *gateways.GatewayConfig
	registeredKafkaConfigs map[uint64]*stream.Stream
}

func (k *kafka) listen(s *stream.Stream, source string) {
	consumer, err := sarama.NewConsumer([]string{s.URL}, nil)
	if err != nil {
		k.gatewayConfig.Log.Error().Str("url", s.URL).Err(err).Msg("failed to connect to cluster")
		return
	}

	topic := s.Attributes[topicKey]
	pString := s.Attributes[partitionKey]
	pInt, err := strconv.ParseInt(pString, 10, 32)
	if err != nil {
		k.gatewayConfig.Log.Error().Str("partition", pString).Err(err).Msg("failed to parse partition key")
		return
	}
	partition := int32(pInt)

	availablePartitions, err := consumer.Partitions(topic)
	if err != nil {
		k.gatewayConfig.Log.Error().Str("topic", topic).Err(err).Msg("unable to get available partitions for kafka topic")
		return
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		k.gatewayConfig.Log.Error().Str("partition", pString).Str("topic", topic).Err(err).Msg("partition does not exist for topic")
		return
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		k.gatewayConfig.Log.Error().Str("partition", pString).Str("topic", topic).Err(err).Msg("failed to create partition consumer for topic")
		return
	}

	// start listening to messages
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			payload, err := gateways.CreateTransformPayload(msg.Value, source)
			if err != nil {
				k.gatewayConfig.Log.Panic().Err(err).Msg("failed to transform event payload")
			}
			k.gatewayConfig.Log.Info().Msg("dispatching the event to gateway-transformer...")
			_, err = http.Post(fmt.Sprintf("http://localhost:%s", k.gatewayConfig.TransformerPort), "application/octet-stream", bytes.NewReader(payload))
			if err != nil {
				k.gatewayConfig.Log.Warn().Err(err).Msg("failed to dispatch the event to gateway-transformer")
			}
		case err := <-partitionConsumer.Errors():
			k.gatewayConfig.Log.Error().Str("partition", pString).Str("topic", topic).Err(err).Msg("received an error")
			return
		}
	}
}

func (k *kafka) RunGateway(cm *apiv1.ConfigMap) error {
	for kConfigkey, kConfigVal := range cm.Data {
		var s *stream.Stream
		err := yaml.Unmarshal([]byte(kConfigVal), &s)
		if err != nil {
			k.gatewayConfig.Log.Error().Str("config", kConfigkey).Err(err).Msg("failed to parse kafka config")
			return err
		}
		k.gatewayConfig.Log.Info().Interface("stream", *s).Msg("kafka configuration")
		key, err := hs.Hash(s, &hs.HashOptions{})
		if err != nil {
			k.gatewayConfig.Log.Warn().Err(err).Msg("failed to get hash of configuration")
			continue
		}
		if _, ok := k.registeredKafkaConfigs[key]; ok {
			k.gatewayConfig.Log.Warn().Interface("config", s).Msg("duplicate configuration")
			continue
		}
		k.registeredKafkaConfigs[key] = s
		go k.listen(s, kConfigkey)
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
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}

	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}

	configName, ok := os.LookupEnv(common.GatewayProcessorConfigMapEnvVar)
	if !ok {
		panic("gateway processor configmap is not provided")
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gatewayConfig := &gateways.GatewayConfig{
		Log:             zlog.New(os.Stdout).With().Logger(),
		Namespace:       namespace,
		Clientset:       clientset,
		TransformerPort: transformerPort,
	}

	k := &kafka{
		gatewayConfig:          gatewayConfig,
		registeredKafkaConfigs: make(map[uint64]*stream.Stream),
	}

	_, err = gatewayConfig.WatchGatewayConfigMap(k, context.Background(), configName)
	if err != nil {
		k.gatewayConfig.Log.Panic().Err(err).Msg("failed to update kafka configuration")
	}

	// run forever
	select {}
}
