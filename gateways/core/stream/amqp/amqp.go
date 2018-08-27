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

	"bytes"
	"context"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/ghodss/yaml"
	hs "github.com/mitchellh/hashstructure"
	zlog "github.com/rs/zerolog"
	amqplib "github.com/streadway/amqp"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
)

const (
	exchangeNameKey = "exchangeName"
	exchangeTypeKey = "exchangeType"
	routingKey      = "routingKey"
	configName      = "amqp-gateway-configmap"
)

type amqp struct {
	gatewayConfig         *gateways.GatewayConfig
	registeredAMQPConfigs map[uint64]*stream.Stream
}

func (a *amqp) RunGateway(cm *apiv1.ConfigMap) error {
	for kConfigkey, kConfigVal := range cm.Data {
		var s *stream.Stream
		err := yaml.Unmarshal([]byte(kConfigVal), &s)
		if err != nil {
			a.gatewayConfig.Log.Error().Str("config", kConfigkey).Err(err).Msg("failed to parse amqp config")
			return err
		}
		a.gatewayConfig.Log.Info().Interface("stream", *s).Msg("amqp configuration")
		key, err := hs.Hash(s, &hs.HashOptions{})
		if err != nil {
			a.gatewayConfig.Log.Warn().Err(err).Msg("failed to get hash of configuration")
			continue
		}
		if _, ok := a.registeredAMQPConfigs[key]; ok {
			a.gatewayConfig.Log.Warn().Interface("config", s).Msg("duplicate configuration")
			continue
		}
		a.registeredAMQPConfigs[key] = s
		go a.listen(s)
	}
	return nil
}

func (a *amqp) listen(s *stream.Stream) {
	conn, err := amqplib.Dial(s.URL)
	if err != nil {
		a.gatewayConfig.Log.Error().Err(err).Msg("failed to connect to server")
		return
	}
	ch, err := conn.Channel()
	if err != nil {
		a.gatewayConfig.Log.Error().Err(err).Msg("failed to open channel")
		return
	}

	delivery, err := getDelivery(ch, s.Attributes)

	if err != nil {
		a.gatewayConfig.Log.Error().Err(err).Msg("failed to get message delivery")
		return
	}

	// start listening for messages
	for {
		select {
		case msg := <-delivery:
			a.gatewayConfig.Log.Info().Msg("received a msg, forwarding it to gateway transformer")
			http.Post(fmt.Sprintf("http://localhost:%s", a.gatewayConfig.TransformerPort), "application/octet-stream", bytes.NewReader(msg.Body))
		}
	}
}

func parseAttributes(attr map[string]string) (string, string, string, error) {
	// parse out the attributes
	exchangeName, ok := attr[exchangeNameKey]
	if !ok {
		return "", "", "", fmt.Errorf("exchange name key is not provided")
	}
	exchangeType, ok := attr[exchangeTypeKey]
	if !ok {
		return exchangeName, "", "", fmt.Errorf("exchange type is not provided")
	}
	routingKey, ok := attr[routingKey]
	if !ok {
		return exchangeName, exchangeType, "", fmt.Errorf("routing key is not provided")
	}
	return exchangeName, exchangeType, routingKey, nil
}

func getDelivery(ch *amqplib.Channel, attr map[string]string) (<-chan amqplib.Delivery, error) {
	exName, exType, rKey, err := parseAttributes(attr)
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(exName, exType, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare %s exchange '%s': %s", exType, exName, err)
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %s", err)
	}

	err = ch.QueueBind(q.Name, rKey, exName, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind %s exchange '%s' to queue with routingKey: %s: %s", exType, exName, rKey, err)
	}

	delivery, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin consuming messages: %s", err)
	}
	return delivery, nil
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
	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gatewayConfig := &gateways.GatewayConfig{
		Log:             zlog.New(os.Stdout).With().Logger(),
		Namespace:       namespace,
		Clientset:       clientset,
		TransformerPort: transformerPort,
	}
	a := &amqp{
		gatewayConfig:         gatewayConfig,
		registeredAMQPConfigs: make(map[uint64]*stream.Stream),
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(a, context.Background(), configName)
	if err != nil {
		a.gatewayConfig.Log.Error().Err(err).Msg("failed to update amqp configuration")
	}

	// run forever
	select {}
}
