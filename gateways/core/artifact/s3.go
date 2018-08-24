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
	"bytes"
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/minio/minio-go"
	zlog "github.com/rs/zerolog"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"k8s.io/apimachinery/pkg/util/wait"
	"github.com/argoproj/argo-events/gateways"
)

const (
	configName = "artifact-gateway-configmap"
)

type s3 struct {
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig *gateways.GatewayConfig
	registeredArtifacts []S3Artifact
}

// S3Artifact contains information about an artifact in S3
type S3Artifact struct {
	S3EventConfig S3EventConfig           `json:"s3EventConfig" protobuf:"bytes,1,opt,name=s3EventConfig"`
	Insecure      bool                    `json:"insecure,omitempty" protobuf:"bytes,2,opt,name=insecure"`
	AccessKey     apiv1.SecretKeySelector `json:"accessKey,omitempty" protobuf:"bytes,3,opt,name=accessKey"`
	SecretKey     apiv1.SecretKeySelector `json:"secretKey,omitempty" protobuf:"bytes,4,opt,name=secretKey"`
}

type S3EventConfig struct {
	Endpoint string                      `json:"endpoint,omitempty" protobuf:"bytes,1,opt,name=endpoint"`
	Bucket   string                      `json:"bucket,omitempty" protobuf:"bytes,2,opt,name=bucket"`
	Region   string                      `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
	Event    minio.NotificationEventType `json:"event,omitempty" protobuf:"bytes,4,opt,name=event"`
	filter   S3Filter                    `json:"filter,omitempty" protobuf:"bytes,5,opt,name=filter"`
}

// S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects
type S3Filter struct {
	Prefix string `json:"prefix" protobuf:"bytes,1,opt,name=prefix"`
	Suffix string `json:"suffix" protobuf:"bytes,2,opt,name=suffix"`
}

// getSecrets retrieves the secret value from the secret in namespace with name and key
func (s *s3) getSecrets(client *kubernetes.Clientset, namespace string, name, key string) (string, error) {
	secretsIf := client.CoreV1().Secrets(namespace)
	var secret *apiv1.Secret
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

func (s *s3) listenToNotifications(artifact *S3Artifact) {
	// retrieve access key id and secret access key
	accessKey, err := s.getSecrets(s.gatewayConfig.Clientset, s.gatewayConfig.Namespace, artifact.AccessKey.Name, artifact.AccessKey.Key)
	if err != nil {
		panic(fmt.Errorf("failed to retrieve access key id %s", artifact.AccessKey.Name))
	}
	secretKey, err := s.getSecrets(s.gatewayConfig.Clientset, s.gatewayConfig.Namespace, artifact.SecretKey.Name, artifact.SecretKey.Key)
	if err != nil {
		panic(fmt.Errorf("failed to retrieve access key id %s", artifact.SecretKey.Name))
	}
	s.gatewayConfig.Log.Info().Str("accesskey", accessKey).Str("secretaccess", secretKey).Msg("minio secs")

	minioClient, err := minio.New(artifact.S3EventConfig.Endpoint, accessKey, secretKey, !artifact.Insecure)
	if err != nil {
		panic(err)
	}

	// Create a done channel to control 'ListenBucketNotification' go routine.
	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	// Listen for bucket notifications
	for notificationInfo := range minioClient.ListenBucketNotification(artifact.S3EventConfig.Bucket, artifact.S3EventConfig.filter.Prefix, artifact.S3EventConfig.filter.Suffix, []string{
		string(artifact.S3EventConfig.Event),
	}, doneCh) {
		if notificationInfo.Err != nil {
			panic(notificationInfo.Err)
		}
		notificationBytes := []byte(fmt.Sprintf("%v", notificationInfo))
		s.gatewayConfig.Log.Info().Msg("forwarding the request")
		http.Post(fmt.Sprintf("http://localhost:%s", s.gatewayConfig.TransformerPort), "application/octet-stream", bytes.NewReader(notificationBytes))
	}
}

func (s *s3) RunGateway(cm *apiv1.ConfigMap) error {
CheckAlreadyRegistered:
	for s3ArtifactKey, s3ArtifactDataStr := range cm.Data {
		var artifact *S3Artifact
		err := yaml.Unmarshal([]byte(s3ArtifactDataStr), &artifact)
		if err != nil {
			// fail silently
			s.gatewayConfig.Log.Warn().Str("artifact", s3ArtifactKey).Err(err).Msg("failed to parse artifact data")
			break
		}
		s.gatewayConfig.Log.Info().Interface("artifact", *artifact).Msg("artifact")
		for _, registeredArtifact := range s.registeredArtifacts {
			if cmp.Equal(registeredArtifact.S3EventConfig, artifact.S3EventConfig) {
				s.gatewayConfig.Log.Warn().Str("bucket", artifact.S3EventConfig.Bucket).Str("event", string(artifact.S3EventConfig.Event)).Msg("event is already registered")
				goto CheckAlreadyRegistered
			}
		}
		s.registeredArtifacts = append(s.registeredArtifacts, *artifact)
		go s.listenToNotifications(artifact)
	}
	return nil
}

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}
	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gatewayConfig := &gateways.GatewayConfig{
		Log: zlog.New(os.Stdout).With().Logger(),
		Namespace: namespace,
		Clientset: clientset,
		TransformerPort: transformerPort,
	}
	s3 := &s3{
		gatewayConfig: gatewayConfig,
		registeredArtifacts: []S3Artifact{},
	}

	_, err = gatewayConfig.WatchGatewayConfigMap(s3, context.Background(), configName)
	if err != nil {
		s3.gatewayConfig.Log.Error().Err(err).Msg("failed to update nats gateway confimap")
	}
	select {}
}
