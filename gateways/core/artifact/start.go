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

package artifact

import (
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	sv1alphav1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
)

// StartConfig runs a configuration
func (ce *S3ConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	defer gateways.CloseChannels(config)

	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("operating on configuration")
	artifact, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *artifact).Msg("artifact configuration")

	go ce.listenToEvents(artifact, config)

	for {
		select {
		case <-config.StartChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
			config.Active = true

		case data := <-config.DataChan:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

// listenEvents listens to minio bucket notifications
func (ce *S3ConfigExecutor) listenToEvents(artifact *S3Artifact, config *gateways.ConfigContext) {
	defer gateways.CloseChannels(config)

	sartifact := &sv1alphav1.S3Artifact{
		Event: artifact.S3EventConfig.Event,
		S3Bucket: &sv1alphav1.S3Bucket{
			Region:    artifact.S3EventConfig.Region,
			Insecure:  artifact.Insecure,
			Bucket:    artifact.S3EventConfig.Bucket,
			Endpoint:  artifact.S3EventConfig.Endpoint,
			SecretKey: artifact.SecretKey,
			AccessKey: artifact.AccessKey,
		},
	}

	creds, err := store.GetCredentials(ce.GatewayConfig.Clientset, ce.GatewayConfig.Namespace, &sv1alphav1.ArtifactLocation{
		S3: sartifact,
	})
	if err != nil {
		config.ErrChan <- err
		return
	}
	minioClient, err := store.NewMinioClient(sartifact, *creds)
	if err != nil {
		config.ErrChan <- err
		return
	}

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	// at this point configuration is successfully running
	config.StartChan <- struct{}{}

	// Listen for bucket notifications
	for notificationInfo := range minioClient.ListenBucketNotification(artifact.S3EventConfig.Bucket, artifact.S3EventConfig.Filter.Prefix, artifact.S3EventConfig.Filter.Suffix, []string{
		string(artifact.S3EventConfig.Event),
	}, config.DoneChan) {
		if notificationInfo.Err != nil {
			config.ErrChan <- notificationInfo.Err
			return
		}
		payload, err := json.Marshal(notificationInfo.Records[0])
		if err != nil {
			config.ErrChan <- err
			return
		}
		config.DataChan <- payload
	}

	config.ShutdownChan <- struct{}{}
}
