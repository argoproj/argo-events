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
	"github.com/ghodss/yaml"
	"github.com/minio/minio-go"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// S3EventSourceExecutor implements Eventing
type S3EventSourceExecutor struct {
	Log zerolog.Logger
	// Clientset is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// S3Artifact contains information about an artifact in S3
// +k8s:openapi-gen=true
type S3Artifact struct {
	// S3EventConfig contains configuration for bucket notification
	S3EventConfig *S3EventConfig `json:"s3EventConfig"`

	// Mode of operation for s3 client
	Insecure bool `json:"insecure,omitempty"`

	// AccessKey
	// +k8s:openapi-gen=false
	AccessKey *corev1.SecretKeySelector `json:"accessKey,omitempty"`

	// SecretKey
	// +k8s:openapi-gen=false
	SecretKey *corev1.SecretKeySelector `json:"secretKey,omitempty"`
}

// S3EventConfig contains configuration for bucket notification
// +k8s:openapi-gen=true
type S3EventConfig struct {
	Endpoint string                      `json:"endpoint,omitempty"`
	Bucket   string                      `json:"bucket,omitempty"`
	Region   string                      `json:"region,omitempty"`
	Event    minio.NotificationEventType `json:"event,omitempty"`
	Filter   S3Filter                    `json:"filter,omitempty"`
}

// S3Filter represents filters to apply to bucket nofifications for specifying constraints on objects
// +k8s:openapi-gen=true
type S3Filter struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

func parseEventSource(config *string) (*S3Artifact, error) {
	var s *S3Artifact
	err := yaml.Unmarshal([]byte(*config), &s)
	if err != nil {
		return nil, err
	}
	return s, err
}
