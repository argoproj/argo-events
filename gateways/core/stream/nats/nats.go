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
	"context"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core"
	"github.com/argoproj/argo-events/gateways/core/stream"
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

// Runs a configuration
func configRunner(config *gateways.ConfigData) error {
	gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("parsing configuration...")

	var s *stream.Stream
	err := yaml.Unmarshal([]byte(config.Config), &s)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to parse configuration")
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Src).Interface("stream", *s).Msg("configuring...")

	conn, err := natsio.Connect(s.URL)
	if err != nil {
		gatewayConfig.Log.Error().Str("url", s.URL).Err(err).Msg("connection failed")
		return err
	}
	gatewayConfig.Log.Debug().Str("server id", conn.ConnectedServerId()).Str("connected url", conn.ConnectedUrl()).
		Str("servers", strings.Join(conn.DiscoveredServers(), ",")).Msg("nats connection")

	var wg sync.WaitGroup
	wg.Add(1)

	// waits till disconnection from client.
	go func() {
		<-config.StopCh
		gatewayConfig.Log.Info().Str("config", config.Src).Msg("stopping the configuration...")
		gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("client disconnected. stopping the configuration...")
		wg.Done()
	}()

	gatewayConfig.Log.Info().Str("config-name", config.Src).Msg("running...")
	config.Active = true

	sub, err := conn.Subscribe(s.Attributes[subjectKey], func(msg *natsio.Msg) {
		gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("dispatching event to gateway-processor")
		gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
			Src: config.Src,
			Payload: msg.Data,
		})
	})
	if err != nil {
		gatewayConfig.Log.Error().Str("url", s.URL).Str("subject", s.Attributes[subjectKey]).Err(err).Msg("failed to subscribe to subject")
	} else {
		gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("configuration is running...")
	}

	wg.Wait()
	err = sub.Unsubscribe()
	if err != nil {
		gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("failed to unsubscribe")
		return err
	}
	return nil
}

func main() {
	gatewayConfig.WatchGatewayConfigMap(context.Background(), configRunner, core.ConfigDeactivator)
	select {}
}
