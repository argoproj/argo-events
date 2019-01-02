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
			gc.markGatewayNodePhase(&status)
		}
	}()

	_, err := gc.WatchGatewayConfigMap(context.Background())
	if err != nil {
		gc.Log.Panic().Err(err).Msg(ErrGatewayConfigmapWatchMsg)
		return err
	}

	port, ok := os.LookupEnv(common.EnvVarGatewayServerPort)
	if ok {
		return fmt.Errorf("port is not provided")
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	RegisterEventingServer(srv, es)

	if err := srv.Serve(lis); err != nil{
		return err
	}
	return nil
}
