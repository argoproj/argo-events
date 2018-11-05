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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	sv1alphav1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
	"encoding/json"
)

// StartConfig runs a configuration
func (s3ce *S3ConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer s3ce.GatewayConfig.GatewayCleanup(config, &errMessage, err)

	s3ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("starting configuration...")
	artifact, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	s3ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *artifact).Msg("artifact configuration")

	go s3ce.listenToEvents(artifact, config)

	for {
		select {
		case <-s3ce.StartChan:
			s3ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
			config.Active = true

		case data := <-s3ce.DataCh:
			s3ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			s3ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src: config.Data.Src,
				Payload: data,
			})

		case <-s3ce.StopChan:
			s3ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping configuration")
			s3ce.DoneCh <- struct{}{}
			config.Active = false
			return nil

		case err := <-s3ce.ErrChan:
			return err
		}
	}
	return nil
}

// listenEvents listens to minio bucket notifications
func (s3ce *S3ConfigExecutor) listenToEvents(artifact *S3Artifact, config *gateways.ConfigContext) {
	defer s3ce.DefaultConfigExecutor.CloseChannels()

	sartifact := &sv1alphav1.S3Artifact{
		Event: artifact.S3EventConfig.Event,
		S3Bucket: &sv1alphav1.S3Bucket{
			Region: artifact.S3EventConfig.Region,
			Insecure: artifact.Insecure,
			Bucket: artifact.S3EventConfig.Bucket,
			Endpoint: artifact.S3EventConfig.Endpoint,
			SecretKey: artifact.SecretKey,
			AccessKey: artifact.AccessKey,
		},
	}

	creds, err := store.GetCredentials(s3ce.GatewayConfig.Clientset, s3ce.GatewayConfig.Namespace,  &sv1alphav1.ArtifactLocation{
		S3: sartifact,
	})
	if err != nil {
		s3ce.ErrChan <- err
	}
	minioClient, err := store.NewMinioClient(sartifact, *creds)
	if err != nil {
		s3ce.ErrChan <- err
	}

	event := s3ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, s3ce.GatewayConfig.Clientset)
	if err != nil {
		s3ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		s3ce.ErrChan <- err
	}

	// at this point configuration is successfully running
	s3ce.StartChan <- struct {}{}

	// Listen for bucket notifications
	for notificationInfo := range minioClient.ListenBucketNotification(artifact.S3EventConfig.Bucket, artifact.S3EventConfig.Filter.Prefix, artifact.S3EventConfig.Filter.Suffix, []string{
		string(artifact.S3EventConfig.Event),
	}, s3ce.DoneCh) {
		if notificationInfo.Err != nil {
			s3ce.ErrChan <- notificationInfo.Err
			return
		}
		payload, err := json.Marshal(notificationInfo.Records[0])
		if err != nil {
			s3ce.ErrChan <- err
			return
		}
		s3ce.DataCh <- payload
	}
	s3ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is stopped")
}
