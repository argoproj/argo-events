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
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/minio/minio-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sync"
	"net/http"
	"github.com/satori/go.uuid"
	"io/ioutil"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
	respBody = `
<PublishResponse xmlns="http://argoevents-sns-server/">
    <PublishResult> 
        <MessageId>` + generateUUID().String() + `</MessageId> 
    </PublishResult> 
    <ResponseMetadata>
       <RequestId>` + generateUUID().String() + `</RequestId>
    </ResponseMetadata> 
</PublishResponse>\n`
)

// s3ConfigExecutor implements ConfigExecutor interface
type s3ConfigExecutor struct{}

// s3EventConfig contains configuration for bucket notification
type s3EventConfig struct {
	Port string
	Endpoint string
	Events   []minio.NotificationEventType
	Filter   S3Filter
}

// S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects
type S3Filter struct {
	Prefix string
	Suffix string
}

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

func generateUUID() uuid.UUID{
	return uuid.NewV4()
}

// StartConfig runs a configuration
func (s3ce *s3ConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")

	var artifact *s3EventConfig
	err = yaml.Unmarshal([]byte(config.Data.Config), &artifact)
	if err != nil {
		errMessage = "failed to parse configuration"
		return err
	}

	gatewayConfig.Log.Debug().Str("config-key", config.Data.Config).Interface("artifact", *artifact).Msg("s3 artifact")

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

	// start a http server listening to notifications from storage grid
	srv := &http.Server{Addr: ":" + fmt.Sprintf("%s", artifact.Port)}
	err = srv.ListenAndServe()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("http server stopped")
	if err == http.ErrServerClosed {
		err = nil
	}
	if err != nil {
		errMessage = "http server stopped"
	}
	if config.Active == true {
		config.StopCh <- struct{}{}
	}

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running...")
	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	http.HandleFunc(artifact.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
		var resp string
		switch request.Method {
		case http.MethodPost, http.MethodPut:
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				gatewayConfig.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to parse request body")
			} else {
				gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("msg", string(body)).Msg("msg body")
			}
		case http.MethodHead:
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("method", http.MethodHead).Msg("received a request")
			resp = ""
		default:
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("method", http.MethodHead).Msg("received a request")
		}
		writer.WriteHeader(http.StatusOK)
		writer.Header().Add("Content-Type", "text/plain")
		writer.Write([]byte(resp))
	})

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

func main() {
	_, err := gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch k8 events for gateway configuration state updates")
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &s3ConfigExecutor{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}

