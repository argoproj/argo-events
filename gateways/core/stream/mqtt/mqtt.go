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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
	"sync"
	"fmt"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// mqttConfigExecutor is
type mqttConfigExecutor struct{}

// StartConfig runs a configuration
func (mce *mqttConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	mqttConfig := config.Data.Config.(*mqtt)
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *mqttConfig).Msg("mqtt configuration")

	var wg sync.WaitGroup
	wg.Add(1)

	// waits till disconnection from client.
	go func() {
		<-config.StopCh
		config.Active = false
		gatewayConfig.Log.Info().Str("config", config.Data.Src).Msg("stopping the configuration...")
		wg.Done()
	}()

	handler := func(c MQTTlib.Client, msg MQTTlib.Message) {
		gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
			Src:     config.Data.Src,
			Payload: msg.Payload(),
		})
	}
	opts := MQTTlib.NewClientOptions().AddBroker(mqttConfig.URL).SetClientID(mqttConfig.ClientId)
	client := MQTTlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		errMessage = "failed to connect to client"
		config.StopCh <- struct{}{}
		err = token.Error()
		return err
	}
	if token := client.Subscribe(mqttConfig.Topic, 0, handler); token.Wait() && token.Error() != nil {
		errMessage = "failed to subscribe to topic"
		err = token.Error()
		return err
	}

	config.Active = true
	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running...")
	wg.Wait()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// StopConfig stops a configuration
func (mce *mqttConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

// Validate validates gateway configuration
func (mce *mqttConfigExecutor) Validate(config *gateways.ConfigContext) error {
	mqttConfig, ok := config.Data.Config.(*mqtt)
	if !ok {
		return gateways.ErrConfigParseFailed
	}
	if mqttConfig.URL == "" {
		return fmt.Errorf("%+v, url must be specified", gateways.ErrInvalidConfig)
	}
	if mqttConfig.Topic == "" {
		return fmt.Errorf("%+v, topic must be specified", gateways.ErrInvalidConfig)
	}
	if mqttConfig.ClientId == "" {
		return fmt.Errorf("%+v, client id must be specified", gateways.ErrInvalidConfig)
	}
	return nil
}

func main() {
	gatewayConfig.StartGateway(&mqttConfigExecutor{})
}
