package main

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/custom/amazon-sns"
)

func main() {
	gc := gateways.NewGatewayConfiguration()
	ce := &amazon_sns.AWSSNSConfigExecutor{}
	ce.GatewayConfig = gc
	gc.StartGateway(ce)
}
