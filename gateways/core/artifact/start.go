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
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/minio/minio-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// getSecrets retrieves the secret value from the secret in namespace with name and key
func getSecrets(client kubernetes.Interface, namespace string, name, key string) (string, error) {
	secretsIf := client.CoreV1().Secrets(namespace)
	var secret *corev1.Secret
	var err error
	_ = wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		secret, err = secretsIf.Get(name, metav1.GetOptions{})
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	val, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("secret '%s' does not have the key '%s'", name, key)
	}
	return string(val), nil
}

// StartConfig runs a configuration
func (ce *S3ConfigExecutor) StartConfig(config *gateways.ConfigContext) {
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
	// retrieve access key id and secret access key
	accessKey, err := getSecrets(ce.Clientset, ce.Namespace, artifact.AccessKey.Name, artifact.AccessKey.Key)
	if err != nil {
		config.ErrChan <- err
		return
	}
	secretKey, err := getSecrets(ce.Clientset, ce.Namespace, artifact.SecretKey.Name, artifact.SecretKey.Key)
	if err != nil {
		config.ErrChan <- err
		return
	}

	minioClient, err := minio.New(artifact.S3EventConfig.Endpoint, accessKey, secretKey, !artifact.Insecure)
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

	ce.Log.Info().Str("config-key", config.Data.Src).Interface("config", artifact).Msg("artifact")

	for {
		select {
		case notification := <-minioClient.ListenBucketNotification(artifact.S3EventConfig.Bucket, artifact.S3EventConfig.Filter.Prefix, artifact.S3EventConfig.Filter.Suffix, []string{
			string(artifact.S3EventConfig.Event),
		}, make(chan struct{})):
			if notification.Err != nil {
				config.ErrChan <- notification.Err
			}
			payload, err := json.Marshal(notification.Records[0])
			if err != nil {
				config.ErrChan <- err
			}
			if err == nil {
				config.DataChan <- payload
			}

		case <-config.DoneChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
			config.ShutdownChan <- struct{}{}
			return
		}
	}
}
