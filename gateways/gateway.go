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

package gateways

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"google.golang.org/grpc"
	"net"
	"os"
	"time"
)

// DispatchEvent dispatches event to gateway transformer for further processing
func (gc *GatewayConfig) DispatchEvent(gatewayEvent *Event) error {
	transformedEvent, err := gc.transformEvent(gatewayEvent)
	if err != nil {
		return err
	}
	switch gc.gw.Spec.DispatchMechanism {
	case v1alpha1.HTTPGateway:
		err = gc.dispatchEventOverHttp(transformedEvent)
		if err != nil {
			return err
		}
	case v1alpha1.NATSGateway:
	default:
		return fmt.Errorf("unknown dispatch mechanism %s", gc.gw.Spec.DispatchMechanism)
	}
	return nil
}

// StartGateway start a gateway
func (gc *GatewayConfig) StartGateway(es EventingServer) error {
	// handle event source's status
	go func() {
		for status := range gc.statusCh {
			gc.updateGatewayResourceState(&status)
		}
	}()

	port, ok := os.LookupEnv(common.EnvVarGatewayServerPort)
	if !ok {
		return fmt.Errorf("port is not provided")
	}

	errCh := make(chan error)

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
		if err != nil {
			errCh <- err
			return
		}
		srv := grpc.NewServer()
		RegisterEventingServer(srv, es)

		if err := srv.Serve(lis); err != nil{
			errCh <- err
		}
	}()

	// wait for server to get started
	time.Sleep(time.Second * 2)

	gc.Log.Info().Msg("gateway started")

	_, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", port))
	if err != nil {
		panic(err)
	}

	gc.Log.Info().Msg("server is up and running")

	go func() {
		_, err := gc.WatchGatewayConfigMap(context.Background())
		if err != nil {
			errCh <- err
		}
	}()

	err = <-errCh
	return err
}
