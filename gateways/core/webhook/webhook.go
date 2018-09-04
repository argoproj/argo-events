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
	"fmt"
	"github.com/argoproj/argo-events/common"
	gwProto "github.com/argoproj/argo-events/gateways/proto"
	"github.com/ghodss/yaml"
	zlog "github.com/rs/zerolog"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"io/ioutil"
)

var (
	hasServerStarted = false
	// as http package does not provide method for unregistering routes,
	// this keeps track of configured http routes and their methods.
	// keeps endpoints as keys and corresponding http methods as a map
	activeRoutes = make(map[string]map[string]gwProto.GatewayExecutor_RunGatewayServer)
	mut          sync.Mutex
)

// hook is a general purpose REST API
type hook struct {
	// REST API endpoint
	Endpoint string `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`

	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method,omitempty" protobuf:"bytes,2,opt,name=method"`

	// Port on which HTTP server is listening for incoming events.
	Port string `json:"port,omitempty" protobuf:"bytes,3,opt,name=port"`
}

// webhook contains gateway configuration and registered endpoints
type webhook struct {
	// serverPort is port on which server is listening
	serverPort string
	// log is json output logger for gateway
	log zlog.Logger
}

// parses webhooks from gateway configuration
func (w *webhook) RunGateway(config *gwProto.GatewayConfig, eventStream gwProto.GatewayExecutor_RunGatewayServer) error {
	w.log.Info().Str("config-name", config.Src).Msg("parsing configuration...")

	var h *hook
	err := yaml.Unmarshal([]byte(config.Config), &h)
	if err != nil {
		w.log.Error().Err(err).Msg("failed to parse configuration")
		return err
	}

	w.log.Info().Interface("config", config.Config).Interface("hook", h).Msg("running configuration")

	if h.Port != "" && !hasServerStarted {
		hasServerStarted = true
		go func() {
			w.log.Info().Str("http-port", h.Port).Msg("http server started listening...")
			log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%s", h.Port), nil))
		}()
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// waits till disconnection from client.
	go func() {
		<-eventStream.Context().Done()
		w.log.Info().Str("config", config.Src).Msg("client disconnected. stopping the configuration...")
		// remove the endpoint and http method configuration.
		mut.Lock()
		activeHTTPMethods := activeRoutes[h.Endpoint]
		delete(activeHTTPMethods, h.Method)
		mut.Unlock()
		wg.Done()
	}()

	// configure endpoint and http method
	if h.Endpoint != "" && h.Method != "" {
		if _, ok := activeRoutes[h.Endpoint]; !ok {
			mut.Lock()
			activeRoutes[h.Endpoint] = make(map[string]gwProto.GatewayExecutor_RunGatewayServer)
			activeRoutes[h.Endpoint][h.Method] = eventStream
			mut.Unlock()
			http.HandleFunc(h.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
				// check if http methods match and route and http method is registered.
				if _, ok := activeRoutes[h.Endpoint]; ok {
					if _, isActive := activeRoutes[h.Endpoint][request.Method]; isActive {
						w.log.Info().Str("endpoint", h.Endpoint).Str("http-method", h.Method).Msg("received a request")
						body, err := ioutil.ReadAll(request.Body)
						if err != nil {
							w.log.Panic().Err(err).Msg("failed to parse request body")
						}

						w.log.Info().Str("endpoint", h.Endpoint).Str("http-method", h.Method).Msg("sending event to gateway processor")
						es := activeRoutes[h.Endpoint][h.Method]
						es.Send(&gwProto.Event{
							Data: body,
						})
					} else {
						w.log.Warn().Str("endpoint", h.Endpoint).Str("http-method", request.Method).Msg("endpoint and http method is not an active route")
						common.SendErrorResponse(writer)
					}
				} else {
					w.log.Warn().Str("endpoint", h.Endpoint).Msg("endpoint is not active")
					common.SendErrorResponse(writer)
				}
			})
		} else {
			mut.Lock()
			activeRoutes[h.Endpoint][h.Method] = eventStream
			mut.Unlock()
		}

		w.log.Info().Str("config-name", config.Src).Msg("configured!!")
		wg.Wait()
	}

	return nil
}

func main() {
	rpcServerPort, ok := os.LookupEnv(common.GatewayProcessorServerPort)
	if !ok {
		panic("gateway rpc server port is not provided")
	}

	w := &webhook{
		log: zlog.New(os.Stdout).With().Logger(),
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", rpcServerPort))
	if err != nil {
		w.log.Fatal().Err(err).Msg("server failed to listen")
	}
	opts := []grpc.ServerOption{}
	grpcServer := grpc.NewServer(opts...)
	gwProto.RegisterGatewayExecutorServer(grpcServer, w)
	w.log.Info().Str("port", rpcServerPort).Msg("gRPC server started listening...")
	grpcServer.Serve(lis)
}
