package main

import (
	"context"
	"github.com/argoproj/argo-events/gateways"
)

func main() {
	gc := gateways.NewGatewayConfiguration()
	// handle event source's status
	go func() {
		for status := range gc.StatusCh {
			gc.UpdateGatewayResourceState(&status)
		}
	}()
	_, err := gc.WatchGatewayConfigMap(context.Background())
	if err != nil {
		panic(err)
	}
	select {}
}
