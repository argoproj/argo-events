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
	"bytes"
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core/stream"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/ghodss/yaml"
	hs "github.com/mitchellh/hashstructure"
	zlog "github.com/rs/zerolog"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
)

const (
	topicKey   = "topic"
	clientID   = "clientID"
)

type mqtt struct {
	gatewayConfig         *gateways.GatewayConfig
	registeredMQTTConfigs map[uint64]*stream.Stream
}

func (m *mqtt) listen(s *stream.Stream, source string) {
	// parse out the attributes
	topic, ok := s.Attributes[topicKey]
	if !ok {
		m.gatewayConfig.Log.Error().Msg("failed to get topic key")
	}

	clientID, ok := s.Attributes[clientID]
	if !ok {
		m.gatewayConfig.Log.Error().Msg("failed to get client id")
	}

	handler := func(c MQTTlib.Client, msg MQTTlib.Message) {
		payload, err := gateways.CreateTransformPayload(msg.Payload(), source)
		if err != nil {
			m.gatewayConfig.Log.Panic().Err(err).Msg("failed to transform event payload")
		}
		m.gatewayConfig.Log.Info().Msg("dispatching event to gateway-transformer...")
		_, err = http.Post(fmt.Sprintf("http://localhost:%s", m.gatewayConfig.TransformerPort), "application/octet-stream", bytes.NewReader(payload))
		if err != nil {
			m.gatewayConfig.Log.Warn().Err(err).Msg("failed to dispatch the event to gateway-transformer")
		}
	}

	opts := MQTTlib.NewClientOptions().AddBroker(s.URL).SetClientID(clientID)
	client := MQTTlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		m.gatewayConfig.Log.Error().Str("client", clientID).Err(token.Error()).Msg("failed to connect to client")
		return
	}
	if token := client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
		m.gatewayConfig.Log.Error().Str("topic", topic).Err(token.Error()).Msg("failed to subscribe to topic")
		return
	}

	m.gatewayConfig.Log.Info().Interface("mqtt-config", s).Msg("started listening to topic")
}

func (m *mqtt) RunGateway(cm *apiv1.ConfigMap) error {
	for mConfigkey, mConfigVal := range cm.Data {
		var s *stream.Stream
		err := yaml.Unmarshal([]byte(mConfigVal), &s)
		if err != nil {
			m.gatewayConfig.Log.Error().Str("config", mConfigkey).Err(err).Msg("failed to parse mqtt config")
			return err
		}
		m.gatewayConfig.Log.Info().Interface("stream", *s).Msg("mqtt configuration")
		key, err := hs.Hash(s, &hs.HashOptions{})
		if err != nil {
			m.gatewayConfig.Log.Warn().Err(err).Msg("failed to get hash of configuration")
			continue
		}
		if _, ok := m.registeredMQTTConfigs[key]; ok {
			m.gatewayConfig.Log.Warn().Interface("config", s).Msg("duplicate configuration")
			continue
		}
		m.registeredMQTTConfigs[key] = s
		go m.listen(s, mConfigkey)
	}
	return nil
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

	m := &mqtt{
		gatewayConfig:         gatewayConfig,
		registeredMQTTConfigs: make(map[uint64]*stream.Stream),
	}

	_, err = gatewayConfig.WatchGatewayConfigMap(m, context.Background(), configName)
	if err != nil {
		m.gatewayConfig.Log.Panic().Err(err).Msg("failed to update mqtt configuration")
	}

	// run forever
	select {}
}
