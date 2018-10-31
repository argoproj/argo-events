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
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	natsio "github.com/nats-io/go-nats"
	"strings"
	"sync"
)

const (
	subjectKey = "subject"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// natsConfigExecutor implements ConfigExecutor
type natsConfigExecutor struct{}

// Runs a configuration
func (nce *natsConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
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
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("client disconnected. stopping the configuration...")
		wg.Done()
	}()

	var s *stream.Stream
	err = yaml.Unmarshal([]byte(config.Data.Config), &s)
	if err != nil {
		errMessage = "failed to parse configuration"
		config.StopCh <- struct{}{}
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("stream", *s).Msg("configuring...")

	conn, err := natsio.Connect(s.URL)
	if err != nil {
		gatewayConfig.Log.Error().Str("url", s.URL).Err(err).Msg("connection failed")
		config.StopCh <- struct{}{}
		return err
	}
	gatewayConfig.Log.Debug().Str("server id", conn.ConnectedServerId()).Str("connected url", conn.ConnectedUrl()).
		Str("servers", strings.Join(conn.DiscoveredServers(), ",")).Msg("nats connection")

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("running...")
	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	sub, err := conn.Subscribe(s.Attributes[subjectKey], func(msg *natsio.Msg) {
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
		gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
			Src:     config.Data.Src,
			Payload: msg.Data,
		})
	})
	if err != nil {
		errMessage = "failed to subscribe to subject"
		config.StopCh <- struct{}{}
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running...")

	wg.Wait()
	err = sub.Unsubscribe()
	if err != nil {
		errMessage = "failed to unsubscribe"
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

func (nce *natsConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

func main() {
	gatewayConfig.StartGateway(&natsConfigExecutor{})
}
