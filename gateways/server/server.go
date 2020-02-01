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

package server

import (
	"fmt"
	"net"
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Channels holds the necessary channels for gateway server to process.
type Channels struct {
	Data chan []byte
	Stop chan struct{}
	Done chan struct{}
}

// NewChannels returns an instance of channels.
func NewChannels() *Channels {
	return &Channels{
		Data: make(chan []byte),
		Stop: make(chan struct{}, 1),
		Done: make(chan struct{}, 1),
	}
}

// StartGateway start a gateway
func StartGateway(es gateways.EventingServer) {
	port, ok := os.LookupEnv(common.EnvVarGatewayServerPort)
	if !ok {
		panic(fmt.Errorf("port is not provided"))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}
	srv := grpc.NewServer()
	gateways.RegisterEventingServer(srv, es)

	fmt.Println("starting gateway server")

	if err := srv.Serve(lis); err != nil {
		panic(err)
	}
}

// Recover recovers from panics in event sources
func Recover(eventSource string) {
	if r := recover(); r != nil {
		fmt.Printf("recovered event source %s from error. recover: %v", eventSource, r)
	}
}

// HandleEventsFromEventSource handles events from the event source.
func HandleEventsFromEventSource(name string, eventStream gateways.Eventing_StartEventSourceServer, channels *Channels, logger *logrus.Logger) {
	for {
		select {
		case data := <-channels.Data:
			logger.WithField(common.LabelEventSource, name).Info("new event received, dispatching to gateway client")
			err := eventStream.Send(&gateways.Event{
				Name:    name,
				Payload: data,
			})
			if err != nil {
				logger.WithField(common.LabelEventSource, name).WithError(err).Errorln("failed to send the event data to the gateway client")
			}

		case <-channels.Stop:
			logger.WithField(common.LabelEventSource, name).Infoln("event source is stopped")
			return

		case <-eventStream.Context().Done():
			logger.WithField(common.LabelEventSource, name).Info("connection is closed by client")
			channels.Done <- struct{}{}
		}
	}
}
