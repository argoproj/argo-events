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
	"net"
	"os"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"k8s.io/apimachinery/pkg/util/wait"
)

func main() {
	// initialize gateway configuration
	gc := gateways.NewGatewayConfiguration()

	serverPort, ok := os.LookupEnv(common.EnvVarGatewayServerPort)
	if !ok {
		panic("gateway server port is not provided")
	}

	// check if gateway server is running
	if err := wait.ExponentialBackoff(wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
		Steps:    30,
	}, func() (bool, error) {
		_, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", serverPort))
		if err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		panic(fmt.Errorf("failed to connect to server on port %s", serverPort))
	}

	// handle event source's status updates
	go func() {
		for status := range gc.StatusCh {
			gc.UpdateGatewayResourceState(&status)
		}
	}()

	// watch updates to gateway resource
	if _, err := gc.WatchGateway(context.Background()); err != nil {
		panic(err)
	}
	// watch for event source updates
	if _, err := gc.WatchEventSources(context.Background()); err != nil {
		panic(err)
	}
	select {}
}
