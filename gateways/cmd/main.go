package main

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"os"
	"time"
)

func main() {
	gc := gateways.NewGatewayConfiguration()

	serverPort, ok := os.LookupEnv(common.EnvVarGatewayServerPort)
	if !ok {
		panic("gateway server port is not provided")
	}

	// check if gateway server is running
	if err := wait.ExponentialBackoff(wait.Backoff{
		Duration: 10 * time.Millisecond,
		Factor: 1.0,
		Jitter: 0.1,
		Steps: 10,
	}, func() (bool, error) {
		_, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", serverPort))
		if err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		panic(fmt.Errorf("failed to connect to server on port %s", serverPort))
	}

	// handle event source's status
	go func() {
		for status := range gc.StatusCh {
			gc.UpdateGatewayResourceState(&status)
		}
	}()
	if _, err := gc.WatchGatewayConfigMap(context.Background()); err != nil {
		panic(err)
	}
	select {}
}
