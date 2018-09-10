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
	"context"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core"
	"github.com/argoproj/argo-events/gateways/core/stream"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/ghodss/yaml"
	"sync"
)

const (
	topicKey = "topic"
	clientID = "clientID"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// Runs a configuration
func configRunner(config *gateways.ConfigData) error {
	gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("parsing configuration...")

	var s *stream.Stream
	err := yaml.Unmarshal([]byte(config.Config), &s)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to parse mqtt config")
		return err
	}
	// parse out the attributes
	topic, ok := s.Attributes[topicKey]
	if !ok {
		gatewayConfig.Log.Error().Msg("failed to get topic key")
	}

	clientID, ok := s.Attributes[clientID]
	if !ok {
		gatewayConfig.Log.Error().Msg("failed to get client id")
	}

	handler := func(c MQTTlib.Client, msg MQTTlib.Message) {
		gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
			Src:     config.Src,
			Payload: msg.Payload(),
		})
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// waits till disconnection from client.
	go func() {
		<-config.StopCh
		gatewayConfig.Log.Info().Str("config", config.Src).Msg("stopping the configuration...")
		wg.Done()
	}()

	opts := MQTTlib.NewClientOptions().AddBroker(s.URL).SetClientID(clientID)
	client := MQTTlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Str("client", clientID).Err(token.Error()).Msg("failed to connect to client")
	}
	if token := client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Str("topic", topic).Err(token.Error()).Msg("failed to subscribe to topic")
	}

	gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("configuration is running...")
	wg.Wait()
	return nil
}

func main() {
	gatewayConfig.WatchGatewayConfigMap(context.Background(), configRunner, core.ConfigDeactivator)
	select {}
}
