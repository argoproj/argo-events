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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	amqplib "github.com/streadway/amqp"
)

const (
	exchangeNameKey = "exchangeName"
	exchangeTypeKey = "exchangeType"
	routingKey      = "routingKey"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// amqpConfigExecutor implements ConfigExecutor interface
type amqpConfigExecutor struct{}

// StartConfig runs a configuration
func (ace *amqpConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("parsing configuration...")

	var s *stream.Stream
	err = yaml.Unmarshal([]byte(config.Data.Config), &s)
	if err != nil {
		errMessage = "failed to parse amqp config"
		return err
	}

	conn, err := amqplib.Dial(s.URL)
	if err != nil {
		errMessage = "failed to connect to server"
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		errMessage = "failed to open channel"
		return err
	}

	delivery, err := getDelivery(ch, s.Attributes)
	if err != nil {
		errMessage = "failed to get message delivery"
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

	// start listening for messages
amqpConfigRunner:
	for {
		select {
		case msg := <-delivery:
			gatewayConfig.Log.Info().Msg("dispatching the event to gateway-transformer")
			gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: msg.Body,
			})
		case <-config.StopCh:
			break amqpConfigRunner
		}
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// StopConfig stops a configuration
func (ace *amqpConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
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
	err := gatewayConfig.TransformerReadinessProbe()
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to connect to gateway transformer")
	}
	_, err = gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch k8 events for gateway configuration state updates")
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &amqpConfigExecutor{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}
