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

package main

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
	"sync"
	"github.com/argoproj/argo-events/gateways/core/artifact/spec"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// s3ConfigExecutor implements ConfigExecutor interface
type s3ConfigExecutor struct{}

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
func (s3ce *s3ConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("starting configuration...")
	artifact := config.Data.Config.(*spec.S3Artifact)
	gatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *artifact).Msg("artifact configuration")

	// retrieve access key id and secret access key
	accessKey, err := getSecrets(gatewayConfig.Clientset, gatewayConfig.Namespace, artifact.AccessKey.Name, artifact.AccessKey.Key)
	if err != nil {
		errMessage = fmt.Sprintf("failed to retrieve access key id %s", artifact.AccessKey.Name)
		return err
	}
	secretKey, err := getSecrets(gatewayConfig.Clientset, gatewayConfig.Namespace, artifact.SecretKey.Name, artifact.SecretKey.Key)
	if err != nil {
		errMessage = fmt.Sprintf("failed to retrieve access key id %s", artifact.SecretKey.Name)
		return err
	}

	gatewayConfig.Log.Debug().Str("accesskey", accessKey).Str("secretaccess", secretKey).Msg("minio secrets")

	minioClient, err := minio.New(artifact.S3EventConfig.Endpoint, accessKey, secretKey, !artifact.Insecure)
	if err != nil {
		errMessage = "failed to get minio client"
		return err
	}

	// Create a done channel to control 'ListenBucketNotification' go routine.
	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	var wg sync.WaitGroup
	wg.Add(1)

	// waits till disconnection from client.
	go func() {
		<-config.StopCh
		config.Active = false
		gatewayConfig.Log.Info().Str("config", config.Data.Src).Msg("stopping the configuration...")
		wg.Done()
	}()

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running...")
	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	// Listen for bucket notifications
	for notificationInfo := range minioClient.ListenBucketNotification(artifact.S3EventConfig.Bucket, artifact.S3EventConfig.Filter.Prefix, artifact.S3EventConfig.Filter.Suffix, []string{
		string(artifact.S3EventConfig.Event),
	}, doneCh) {
		if notificationInfo.Err != nil {
			err = notificationInfo.Err
			errMessage = "notification error"
			config.StopCh <- struct{}{}
			break
		} else {
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			payload, err := json.Marshal(notificationInfo.Records[0])
			if err != nil {
				errMessage = err.Error()
				config.StopCh <- struct{}{}
				break
			}
			gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: []byte(string(payload)),
			})
		}
	}
	wg.Wait()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// StopConfig stops the configuration
func (s3ce *s3ConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

// Validate validates s3 configuration
func(s3ce *s3ConfigExecutor) Validate(config *gateways.ConfigContext) error {
	artifact, ok := config.Data.Config.(*spec.S3Artifact)
	if !ok {
		return gateways.ErrConfigParseFailed
	}
	if artifact.AccessKey != nil {
		return fmt.Errorf("access key can't be empty")
	}
	if artifact.SecretKey != nil {
		return fmt.Errorf("secret key can't be empty")
	}
	if artifact.S3EventConfig != nil {
		return fmt.Errorf("s3 event configuration can't be empty")
	}
	if artifact.S3EventConfig.Endpoint == "" {
		return fmt.Errorf("endpoint url can't be empty")
	}
	if artifact.S3EventConfig.Bucket == "" {
		return fmt.Errorf("bucket name can't be empty")
	}
	if artifact.S3EventConfig.Event != "" && minio.NotificationEventType(artifact.S3EventConfig.Event) == "" {
		return fmt.Errorf("unknown event %s", artifact.S3EventConfig.Event)
	}
	return nil
}

func main() {
	gatewayConfig.StartGateway(&s3ConfigExecutor{})
}
