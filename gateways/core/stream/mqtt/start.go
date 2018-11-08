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

package mqtt

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
)

// StartConfig runs a configuration
func (ce *MqttConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	m, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- err
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *m).Msg("mqtt configuration")

	go ce.listenEvents(m, config)

	for {
		select {
		case <-config.StartChan:
			config.Active = true
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running")

		case data := <-config.DataChan:
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *MqttConfigExecutor) listenEvents(m *mqtt, config *gateways.ConfigContext) {
	handler := func(c MQTTlib.Client, msg MQTTlib.Message) {
		config.DataChan <- msg.Payload()
	}
	opts := MQTTlib.NewClientOptions().AddBroker(m.URL).SetClientID(m.ClientId)
	client := MQTTlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		config.ErrChan <- token.Error()
		return
	}
	if token := client.Subscribe(m.Topic, 0, handler); token.Wait() && token.Error() != nil {
		config.ErrChan <- token.Error()
		return
	}

	config.Active = true
	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err := common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	config.StartChan <- struct{}{}

	<-config.DoneChan
	token := client.Unsubscribe(m.Topic)
	if token.Error() != nil {
		ce.GatewayConfig.Log.Error().Err(token.Error()).Str("config-key", config.Data.Src).Msg("failed to unsubscribe client")
	}
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
	config.ShutdownChan <- struct{}{}
}
