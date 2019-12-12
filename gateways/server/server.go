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
func HandleEventsFromEventSource(name string, eventStream gateways.Eventing_StartEventSourceServer, dataCh chan []byte, errorCh chan error, doneCh chan struct{}, log *logrus.Logger) error {
	for {
		select {
		case data := <-dataCh:
			log.WithField(common.LabelEventSource, name).Info("new event received, dispatching to gateway client")
			err := eventStream.Send(&gateways.Event{
				Name:    name,
				Payload: data,
			})
			if err != nil {
				return err
			}

		case err := <-errorCh:
			log.WithField(common.LabelEventSource, name).WithError(err).Error("error occurred processing the event source")
			return err

		case <-eventStream.Context().Done():
			log.WithField(common.LabelEventSource, name).Info("connection is closed by client")
			doneCh <- struct{}{}
			return nil
		}
	}
}
