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

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	amqplib "github.com/streadway/amqp"
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

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	amqpConfig := config.Data.Config.(*amqp)
	conn, err := amqplib.Dial(amqpConfig.URL)
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

// Validate validates gateway configuration
func (ace *amqpConfigExecutor) Validate(config *gateways.ConfigContext) error {
	return nil
}

func getDelivery(ch *amqplib.Channel, config *amqp) (<-chan amqplib.Delivery, error) {
	err := ch.ExchangeDeclare(config.ExchangeName, config.ExchangeType, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange with name %s and type %s. err: %+v", config.ExchangeName, config.ExchangeType, err)
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %s", err)
	}

	err = ch.QueueBind(q.Name, config.RoutingKey, config.ExchangeName, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind %s exchange '%s' to queue with routingKey: %s: %s", config.ExchangeType, config.ExchangeName, config.RoutingKey, err)
	}

	delivery, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin consuming messages: %s", err)
	}
	return delivery, nil
}

func main() {
	gatewayConfig.StartGateway(&amqpConfigExecutor{})
}
