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

	"context"
	gateways "github.com/argoproj/argo-events/gateways/core"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/ghodss/yaml"
	amqplib "github.com/streadway/amqp"
)

const (
	exchangeNameKey = "exchangeName"
	exchangeTypeKey = "exchangeType"
	routingKey      = "routingKey"
)

type amqp struct {
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig *gateways.GatewayConfig
}

func (a *amqp) RunConfiguration(config *gateways.ConfigData) error {
	a.gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("parsing configuration...")

	var s *stream.Stream
	err := yaml.Unmarshal([]byte(config.Config), &s)
	if err != nil {
		a.gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to parse amqp config")
		return err
	}

	conn, err := amqplib.Dial(s.URL)
	if err != nil {
		a.gatewayConfig.Log.Error().Err(err).Msg("failed to connect to server")
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		a.gatewayConfig.Log.Error().Err(err).Msg("failed to open channel")
		return err
	}

	delivery, err := getDelivery(ch, s.Attributes)

	if err != nil {
		a.gatewayConfig.Log.Error().Err(err).Msg("failed to get message delivery")
		return err
	}

	a.gatewayConfig.Log.Info().Str("config-name", config.Src).Msg("running...")
	config.Active = true
	// start listening for messages
amqpConfigRunner:
	for {
		select {
		case msg := <-delivery:
			a.gatewayConfig.Log.Info().Msg("dispatching the event to gateway-transformer")
			a.gatewayConfig.DispatchEvent(msg.Body, config.Src)
		case <-config.StopCh:
			break amqpConfigRunner
		}
	}
	return nil
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
	gatewayConfig := gateways.NewGatewayConfiguration()
	a := &amqp{
		gatewayConfig,
	}
	gatewayConfig.WatchGatewayConfigMap(a, context.Background())
	select {}
}
