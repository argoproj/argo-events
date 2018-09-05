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
	gateways "github.com/argoproj/argo-events/gateways/core"
	"github.com/argoproj/argo-events/gateways/core/stream"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/ghodss/yaml"
	zlog "github.com/rs/zerolog"
	"os"
	"sync"
)

const (
	topicKey = "topic"
	clientID = "clientID"
)

type mqtt struct {
	// log is json output logger for gateway
	log zlog.Logger
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig *gateways.GatewayConfig
}

func (m *mqtt) RunConfiguration(config *gateways.ConfigData) error {
	m.log.Info().Str("config-key", config.Src).Msg("parsing configuration...")

	var s *stream.Stream
	err := yaml.Unmarshal([]byte(config.Config), &s)
	if err != nil {
		m.log.Error().Str("config-key", config.Src).Err(err).Msg("failed to parse mqtt config")
		return err
	}
	// parse out the attributes
	topic, ok := s.Attributes[topicKey]
	if !ok {
		m.log.Error().Msg("failed to get topic key")
	}

	clientID, ok := s.Attributes[clientID]
	if !ok {
		m.log.Error().Msg("failed to get client id")
	}

	handler := func(c MQTTlib.Client, msg MQTTlib.Message) {
		m.gatewayConfig.DispatchEvent(msg.Payload(), config.Src)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// waits till disconnection from client.
	go func() {
		<-config.StopCh
		m.log.Info().Str("config", config.Src).Msg("stopping the configuration...")
		wg.Done()
	}()

	opts := MQTTlib.NewClientOptions().AddBroker(s.URL).SetClientID(clientID)
	client := MQTTlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		m.log.Error().Str("config-key", config.Src).Str("client", clientID).Err(token.Error()).Msg("failed to connect to client")
	}
	if token := client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
		m.log.Error().Str("config-key", config.Src).Str("topic", topic).Err(token.Error()).Msg("failed to subscribe to topic")
	}

	m.log.Info().Str("config-key", config.Src).Msg("running...")
	wg.Wait()
	return nil
}

func main() {
	a := &mqtt{
		log:           zlog.New(os.Stdout).With().Logger(),
		gatewayConfig: gateways.NewGatewayConfiguration(),
	}
	a.gatewayConfig.WatchGatewayConfigMap(a, context.Background())
	select {}
}
