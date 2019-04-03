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
	"github.com/argoproj/argo-events/pkg/apis/gateway"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	zlog "github.com/rs/zerolog"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net"
	"os"
	"runtime/debug"
)

// StartGateway start a gateway
func StartGateway(es EventingServer) {
	port, ok := os.LookupEnv(common.EnvVarGatewayServerPort)
	if !ok {
		panic(fmt.Errorf("port is not provided"))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}
	srv := grpc.NewServer()
	RegisterEventingServer(srv, es)

	fmt.Println("starting gateway server")

	if err := srv.Serve(lis); err != nil {
		panic(err)
	}
}

// Recover recovers from panics in event sources
func Recover(eventSource string) {
	if r := recover(); r != nil {
		fmt.Printf("recovered event source %s from error. recover: %v", eventSource, r)
		debug.PrintStack()
	}
}

// HandleEventsFromEventSource handles events from the event source.
func HandleEventsFromEventSource(name string, eventStream Eventing_StartEventSourceServer, dataCh chan []byte, errorCh chan error, doneCh chan struct{}, log *zlog.Logger) error {
	for {
		select {
		case data := <-dataCh:
			log.Info().Str("event-source-name", name).Msg("new event received, dispatching to gateway client")
			err := eventStream.Send(&Event{
				Name:    name,
				Payload: data,
			})
			if err != nil {
				return err
			}

		case err := <-errorCh:
			log.Info().Str("event-source-name", name).Err(err).Msg("error occurred while getting event from event source")
			return err

		case <-eventStream.Context().Done():
			log.Info().Str("event-source-name", name).Msg("connection is closed by client")
			doneCh <- struct{}{}
			return nil
		}
	}
}

// WatchGateway watches updates to gateway resource
func (gc *GatewayConfig) WatchGateway() error {
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan InformerResult)

	done := ctx.Done()

	go func() {
		for {
			select {
			case result := <-resultCh:
				if result.Type == UPDATE {
					var g *v1alpha1.Gateway
					gwBytes, err := yaml.Marshal(result.Obj)
					if err != nil {
						gc.Log.Error().Err(err).Msg("failed to marshal updated object")
						continue
					}

					if err = yaml.Unmarshal(gwBytes, &g); err != nil {
						gc.Log.Error().Err(err).Msg("failed to unmarshal updated object")
						continue
					}

					gc.Log.Info().Interface("gateway", *g).Msg("gatewayyyysyasyyas")

					gc.Log.Info().Msg("detected gateway update.")
					gc.StatusCh <- EventSourceStatus{
						Phase:   v1alpha1.NodePhaseResourceUpdate,
						Gw:      g,
						Message: "gateway_resource_update",
					}
				}
			case <-done:
				break
			}
		}
	}()

	gvr := schema.GroupVersionResource{
		Group:    gateway.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: gateway.Plural,
	}
	if err := gc.WatchResource(ctx, gc.KubeConfig, gvr, gc.Namespace, nil, map[string]string{"metadata.name": gc.Name}, 0, resultCh); err != nil {
		cancel()
		panic(err)
	}
	return nil
}

// WatchGatewayEventSource watches updates to gateway specific custom resource that holds event source configurations
func (gc *GatewayConfig) WatchGatewayEventSource() error {
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan InformerResult)

	done := ctx.Done()

	go func() {
		for {
			select {
			case result := <-resultCh:
				gc.Log.Info().Msg("detected gateway event source update.")
				if err := gc.manageEventSources(result.Obj); err != nil {
					gc.Log.Error().Err(err).Msg("failed to manage event sources")
				}
			case <-done:
				return
			}
		}
	}()

	gvr := schema.GroupVersionResource{
		Group:    gc.gw.Spec.EventSource.Group,
		Version:  gc.gw.Spec.EventSource.Version,
		Resource: gc.gw.Spec.EventSource.Resource,
	}

	if err := gc.WatchResource(ctx, gc.KubeConfig, gvr, gc.Namespace, nil, map[string]string{"metadata.name": gc.gw.Spec.EventSource.Name}, 0, resultCh); err != nil {
		cancel()
		panic(err)
	}

	return nil
}
