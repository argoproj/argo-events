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

package google_pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
)

func (ce *GCPPubSubConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")
	gps, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
		return
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *gps).Msg("gcp pubsub configuration")

	go ce.listenEvents(gps, config)

	for {
		select {
		case _, ok := <-config.StartChan:
			if ok {
				ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
				config.Active = true
			}

		case data, ok := <-config.DataChan:
			if ok {
				err := ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: data,
				})
				if err != nil {
					config.ErrChan <- err
				}
			}

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *GCPPubSubConfigExecutor) listenEvents(gps *GCPPubSubConfig, config *gateways.ConfigContext) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, gps.ProjectId)
	if err != nil {
		config.ErrChan <- err
		return
	}

	subscription := client.Subscription(gps.Subscription)
	exists, err := subscription.Exists(ctx)
	if err != nil {
		config.ErrChan <- err
		return
	}

	if !exists {
		topic := client.Topic(gps.Topic)
		subscription, err = client.CreateSubscription(ctx, gps.Subscription, pubsub.SubscriptionConfig{
			Topic: topic,
		})
		if err != nil {
			config.ErrChan <- err
			return
		}
	}

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("k8 event created marking configuration as running")

	config.StartChan <- struct{}{}

	err = subscription.Receive(ctx, func(ctx context.Context, message *pubsub.Message) {
		config.DataChan <- message.Data
	})

	if err != nil {
		config.ErrChan <- err
	}

	<-config.DoneChan
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
	config.ShutdownChan <- struct{}{}
}
