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
	"strings"
	"sync"

	"github.com/argoproj/argo-events/sdk"
	"github.com/argoproj/argo-events/shared"
	"go.uber.org/zap"
)

// SignalManager helps manage the various signals microservices
// TODO: clients may become disconnected during operations so we need a way to remove bad clients
// this can be accomplished through a health-checking routine
type SignalManager struct {
	microClient *shared.MicroSignalClient
	sync.Mutex
	clients map[string]sdk.SignalClient
	log     *zap.SugaredLogger
}

// NewSignalManager creates a new SignalManager
func NewSignalManager(log *zap.SugaredLogger) (*SignalManager, error) {
	mgr := SignalManager{
		microClient: shared.NewMicroSignalClient(),
		clients:     make(map[string]sdk.SignalClient),
		log:         log,
	}
	// TODO: add to cache of the builtin signal services
	return &mgr, nil
}

// Dispense the signal client with the given name
// NOTE: assumes the name matches the service name
func (pm *SignalManager) Dispense(name string) (sdk.SignalClient, error) {
	pm.Lock()
	defer pm.Unlock()
	lowercase := strings.ToLower(name)
	//client, ok := pm.clients[lowercase]
	//if !ok {
	c := pm.microClient.NewSignalService(lowercase)
	pm.log.Debugf("Dispensed '%s' signal service", lowercase)
	pm.clients[name] = c
	return c, nil
	//}
	//return client, nil
}
