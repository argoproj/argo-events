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
	"bytes"
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"time"
)

// TransformerReadinessProbe checks whether gateway transformer is running or not
func (gc *GatewayConfig) TransformerReadinessProbe() error {
	return wait.ExponentialBackoff(wait.Backoff{
		Steps:    5,
		Duration: 1 * time.Minute,
		Factor:   1.0,
		Jitter:   0.1,
	}, func() (bool, error) {
		_, err := http.Get(fmt.Sprintf("http://localhost:%s/readiness", gc.transformerPort))
		if err != nil {
			return false, err
		}
		return true, nil
	})
}

// DispatchEvent dispatches event to gateway transformer for further processing
func (gc *GatewayConfig) DispatchEvent(gatewayEvent *GatewayEvent) error {
	payload, err := TransformerPayload(gatewayEvent.Payload, gatewayEvent.Src)
	if err != nil {
		gc.Log.Warn().Str("config-key", gatewayEvent.Src).Err(err).Msg("failed to transform request body.")
		return err
	}
	gc.Log.Info().Str("config-key", gatewayEvent.Src).Msg("dispatching the event to gateway-transformer...")

	_, err = http.Post(fmt.Sprintf("http://localhost:%s", gc.transformerPort), "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		gc.Log.Warn().Str("config-key", gatewayEvent.Src).Err(err).Msg("failed to dispatch event to gateway-transformer.")
		return err
	}
	return nil
}

// StartGateway starts a gateway
func (gc *GatewayConfig) StartGateway(configExecutor ConfigExecutor) error {
	err := gc.TransformerReadinessProbe()
	if err != nil {
		gc.Log.Panic().Err(err).Msg(ErrGatewayTransformerConnectionMsg)
		return err
	}
	_, err = gc.WatchGatewayEvents(context.Background())
	if err != nil {
		gc.Log.Panic().Err(err).Msg(ErrGatewayEventWatchMsg)
		return err
	}
	_, err = gc.WatchGatewayConfigMap(context.Background(), configExecutor)
	if err != nil {
		gc.Log.Panic().Err(err).Msg(ErrGatewayConfigmapWatchMsg)
		return err
	}
	select {}
}
