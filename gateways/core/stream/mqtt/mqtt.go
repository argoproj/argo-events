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
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
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

// mqttConfigExecutor is
type mqttConfigExecutor struct{}

// StartConfig runs a configuration
func (mce *mqttConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("parsing configuration...")
	var wg sync.WaitGroup
	wg.Add(1)

	// waits till disconnection from client.
	go func() {
		<-config.StopCh
		config.Active = false
		gatewayConfig.Log.Info().Str("config", config.Data.Src).Msg("stopping the configuration...")
		wg.Done()
	}()

	var s *stream.Stream
	err = yaml.Unmarshal([]byte(config.Data.Config), &s)
	if err != nil {
		errMessage = "failed to parse mqtt config"
		config.StopCh <- struct{}{}
		return err
	}
	// parse out the attributes
	topic, ok := s.Attributes[topicKey]
	if !ok {
		errMessage = "failed to get topic key"
		err = fmt.Errorf(errMessage)
		return err
	}

	clientID, ok := s.Attributes[clientID]
	if !ok {
		errMessage = "failed to get client id"
		err = fmt.Errorf(errMessage)
		return err
	}

	handler := func(c MQTTlib.Client, msg MQTTlib.Message) {
		gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
			Src:     config.Data.Src,
			Payload: msg.Payload(),
		})
	}
	opts := MQTTlib.NewClientOptions().AddBroker(s.URL).SetClientID(clientID)
	client := MQTTlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		errMessage = "failed to connect to client"
		config.StopCh <- struct{}{}
		err = token.Error()
		return err
	}
	if token := client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
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

func main() {
	err := gatewayConfig.TransformerReadinessProbe()
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to connect to gateway transformer")
	}
	_, err = gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch k8 events for gateway configuration state updates")
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &mqttConfigExecutor{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}
