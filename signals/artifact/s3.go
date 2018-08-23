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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"os"
	"k8s.io/apimachinery/pkg/util/wait"
)

type s3 struct {
	clientset           *kubernetes.Clientset
	config              string
	namespace           string
	transformerPort     string
	registeredArtifacts []S3Artifact
	log                 zlog.Logger
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
	accessKey, err := s.getSecrets(s.clientset, s.namespace, artifact.AccessKey.Name, artifact.AccessKey.Key)
	if err != nil {
		panic(fmt.Errorf("failed to retrieve access key id %s", artifact.AccessKey))
	}
	secretKey, err := s.getSecrets(s.clientset, s.namespace, artifact.SecretKey.Name, artifact.SecretKey.Key)
	if err != nil {
		panic(fmt.Errorf("failed to retrieve access key id %s", artifact.SecretKey))
	}
	s.log.Info().Str("accesskey", accessKey).Str("secretaccess", secretKey).Msg("minio secs")

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
		s.log.Info().Msg("forwarding the request")
		http.Post(fmt.Sprintf("http://localhost:%s", s.transformerPort), "application/octet-stream", bytes.NewReader(notificationBytes))
	}
}

func (s *s3) WatchGatewayTransformerConfigMap(ctx context.Context, name string) (cache.Controller, error) {
	source := s.newStoreConfigMapWatch(name)
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					s.log.Info().Str("config-map", name).Msg("detected ConfigMap update. Updating the controller config.")
					err := s.updateConfig(cm)
					if err != nil {
						s.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					s.log.Info().Msg("detected ConfigMap update. Updating the controller config.")
					err := s.updateConfig(newCm)
					if err != nil {
						s.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (s *s3) newStoreConfigMapWatch(name string) *cache.ListWatch {
	x := s.clientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(s.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(s.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (s *s3) updateConfig(cm *apiv1.ConfigMap) error {
CheckAlreadyRegistered:
	for s3ArtifactKey, s3ArtifactDataStr := range cm.Data {
		var artifact *S3Artifact
		err := yaml.Unmarshal([]byte(s3ArtifactDataStr), &artifact)
		if err != nil {
			// fail silently
			s.log.Warn().Str("artifact", s3ArtifactKey).Err(err).Msg("failed to parse artifact data")
			break
		}
		s.log.Info().Interface("artifact", *artifact).Msg("artifact")
		for _, registeredArtifact := range s.registeredArtifacts {
			if cmp.Equal(registeredArtifact.S3EventConfig, artifact.S3EventConfig) {
				s.log.Warn().Str("bucket", artifact.S3EventConfig.Bucket).Str("event", string(artifact.S3EventConfig.Event)).Msg("event is already registered")
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

	// Todo: hardcoded for now, move it to constants
	config := "artifact-gateway-configmap"
	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}
	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)
	s3 := &s3{
		registeredArtifacts: []S3Artifact{},
		namespace:           namespace,
		log:                 zlog.New(os.Stdout).With().Logger(),
		config:              config,
		clientset:           clientset,
		transformerPort:     transformerPort,
	}

	_, err = s3.WatchGatewayTransformerConfigMap(context.Background(), config)

	select {}
}
