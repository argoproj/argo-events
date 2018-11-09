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
	"os"
)

const (
	serverAddr = "localhost:%s"
)

// grpcConfigExecutor implements ConfigExecutor
type grpcConfigExecutor struct{
	*gateways.GatewayConfig
}

var (
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
func (ce *grpcConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithWaitForHandshake(),
	}

	conn, err := grpc.Dial(fmt.Sprintf(serverAddr, rpcServerPort), opts...)
	if err != nil {
		config.ErrChan <- err
		return
	}

	// create a client for gateway server
	client := gwProto.NewGatewayExecutorClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	config.Cancel = cancel

	eventStream, err := client.StartConfig(ctx, &gwProto.GatewayConfig{
		Config: config.Data.Config,
		Src:    config.Data.Src,
	})
	if err != nil {
		config.ErrChan <- err
		return
	}

	ce.Log.Info().Str("config-key", config.Data.Src).Msg("connected with server and got a event stream")

	event := ce.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	k8Event, err := common.CreateK8Event(event, ce.Clientset)
	if err != nil {
		ce.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	ce.Log.Info().Str("config-key", config.Data.Src).Str("phase", string(v1alpha1.NodePhaseRunning)).Str("event-name", k8Event.Name).Msg("k8 event created")
	for {
		event, err := eventStream.Recv()
		if err != nil {
			ce.Log.Info().Str("config-key", config.Data.Src).Msg("event stream stopped")
			config.ErrChan <- err
			return
		}
		// event should never be nil
		if event == nil {
			config.ErrChan <- fmt.Errorf("event can't be nil")
			return
		}
		ce.DispatchEvent(&gateways.GatewayEvent{
			Src:     config.Data.Src,
			Payload: event.Data,
		})
	}
}

// Stop config disconnects the client connection and hence stops the configuration
func (gce *grpcConfigExecutor) StopConfig(config *gateways.ConfigContext) {
	config.Cancel()
}

func (gce *grpcConfigExecutor) Validate(config *gateways.ConfigContext) error {
	return nil
}

func main() {
	gc := gateways.NewGatewayConfiguration()
	ce := &grpcConfigExecutor{}
	ce.GatewayConfig = gc
	gc.StartGateway(ce)
}
