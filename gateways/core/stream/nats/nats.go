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
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/ghodss/yaml"
	natsio "github.com/nats-io/go-nats"
	"strings"
	"sync"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/argoproj/argo-events/gateways/core"
	"context"
)

const (
	subjectKey = "subject"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// Runs a configuration
func configRunner(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, errMessage, err)

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

	event := gatewayConfig.K8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data.Src)
	err = gatewayConfig.CreateK8Event(event)
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
	} else {
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running...")
	}

	wg.Wait()
	err = sub.Unsubscribe()
	if err != nil {
		errMessage = "failed to unsubscribe"
		return err
	}
	return nil
}

func main() {
	_, err := gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch k8 events for gateway state updates")
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), configRunner, core.ConfigDeactivator)
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}
