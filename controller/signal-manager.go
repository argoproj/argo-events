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

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argoproj/argo-events/sdk"
	"github.com/argoproj/argo-events/shared"
)

// SignalManager helps manage the various signals microservices
// TODO: clients may become disconnected during operations so we need a way to remove bad clients
// this can be accomplished through a health-checking routine
type SignalManager struct {
	microClient *shared.MicroSignalClient
}

// NewSignalManager creates a new SignalManager
// This produces an error on 2 conditions:
// 1. the microClient fails in listing services
// 2. there are no registered signal services
func NewSignalManager() (*SignalManager, error) {
	c := shared.NewMicroSignalClient()
	services, err := c.DiscoverSignals()
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return nil, fmt.Errorf("signal manager found 0 registered signals! you must deploy a registered Micro signal service. see: https://github.com/argoproj/argo-events/blob/master/docs/quickstart.md for getting started")
	}
	return &SignalManager{microClient: c}, nil
}

// Dispense the signal client with the given name
// 3 conditions apply which generate errors
// 1. the microClient fails in listing services
// 2. the name is not a discoverable signal service
// 3. the service client ping fails
func (pm *SignalManager) Dispense(name string) (sdk.SignalClient, error) {
	services, err := pm.microClient.DiscoverSignals()
	if err != nil {
		return nil, err
	}
	lowercase := strings.ToLower(name)
	if !contains(services, lowercase) {
		return nil, fmt.Errorf("the signal '%s' does not exist with the signal universe. please choose one from: %s", lowercase, services)
	}
	c := pm.microClient.NewSignalService(lowercase)
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	return c, c.Ping(ctx)
}
