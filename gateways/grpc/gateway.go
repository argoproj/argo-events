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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"google.golang.org/grpc"
	"io"
	"os"
)

const (
	serverAddr = "localhost:%s"
)

// grpcConfigExecutor implements ConfigExecutor
type grpcConfigExecutor struct{}

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

// StartConfig establishes connection with gateway server and sends new configurations to run.
// also disconnect the clients for stale configurations
func (gce *grpcConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithWaitForHandshake(),
	}

	conn, err := grpc.Dial(fmt.Sprintf(serverAddr, rpcServerPort), opts...)
	if err != nil {
		errMessage = "failed to dial"
		return err
	}

	// create a client for gateway server
	client := gwProto.NewGatewayExecutorClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	config.Cancel = cancel

	eventStream, err := client.RunGateway(ctx, &gwProto.GatewayConfig{
		Config: config.Data.Config,
		Src:    config.Data.Src,
	})
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("connected with server and got a event stream")

	if err != nil {
		errMessage = "failed to get event stream"
		return err
	}

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	k8Event, err := common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("phase", string(v1alpha1.NodePhaseRunning)).Str("event-name", k8Event.Name).Msg("k8 event created")
	for {
		event, err := eventStream.Recv()
		if err == io.EOF {
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("event stream stopped")
			return nil
		}
		if err != nil {
			errMessage = "failed to receive events on stream"
			return err
		}
		// event should never be nil
		if event == nil {
			errMessage = "event can't be nil"
			err = fmt.Errorf(errMessage)
			return err
		} else {
			gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: event.Data,
			})
		}
	}
	return nil
}

// Stop config disconnects the client connection and hence stops the configuration
func (gce *grpcConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	config.Cancel()
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
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &grpcConfigExecutor{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}
