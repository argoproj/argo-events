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
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwProto "github.com/argoproj/argo-events/gateways/grpc/proto"
	"google.golang.org/grpc"
	"io"
	"os"
)

const (
	serverAddr = "localhost:%s"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
	rpcServerPort = func() string {
		rpcServerPort, ok := os.LookupEnv(common.GatewayProcessorGRPCServerPort)
		if !ok {
			panic("gateway rpc server port is not provided")
		}
		return rpcServerPort
	}()
)

// runConfig establishes connection with gateway server and sends new configurations to run.
// also disconnect the clients for stale configurations
func configRunner(config *gateways.ConfigData) error {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithWaitForHandshake(),
	}

	conn, err := grpc.Dial(fmt.Sprintf(serverAddr, rpcServerPort), opts...)
	if err != nil {
		gatewayConfig.Log.Fatal().Str("config-key", config.Src).Err(err).Msg("failed to dial")
	}

	// create a client for gateway server
	client := gwProto.NewGatewayExecutorClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	config.Cancel = cancel

	eventStream, err := client.RunGateway(ctx, &gwProto.GatewayConfig{
		Config: config.Config,
		Src:    config.Src,
	})
	gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("connected with server and got a event stream")

	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to get event stream")
		return err
	}
	for {
		event, err := eventStream.Recv()
		if err == io.EOF {
			gatewayConfig.Log.Info().Str("config-key", config.Src).Msg("event stream stopped")
			return err
		}
		if err != nil {
			gatewayConfig.Log.Warn().Str("config-key", config.Src).Err(err).Msg("failed to receive events on stream")
			return err
		}
		// event should never be nil
		if event == nil {
			gatewayConfig.Log.Warn().Str("config-key", config.Src).Err(err).Msg("event can't be nil")
		} else {
			gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src: config.Src,
				Payload: event.Data,
			})
		}
	}
	return nil
}

func configDeactivator(config *gateways.ConfigData) error {
	config.Cancel()
	return nil
}

func main() {
	gatewayConfig.WatchGatewayConfigMap(context.Background(), configRunner, configDeactivator)
	select {}
}
