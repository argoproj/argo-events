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

package aws_sqs

import (
	"github.com/AdRoll/goamz/aws"
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
)

// AWSSQSConfig contains information to listen to AWS SQS
type AWSSQSConfig struct {
	// URL is queue url
	URL string `json:"url"`

	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey"`

	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey"`

	// Region is AWS region
	Region string `json:"region"`

	// Queue is AWS SQS queue to listen to for messages
	Queue string `json:"queue"`

	// Frequency is polling frequency
	Frequency string `json:"frequency"`
}

// AWSSQSConfigExecutor implements ConfigExecutor
type AWSSQSConfigExecutor struct {
	*gateways.GatewayConfig
	auth aws.Auth
}

func parseConfig(config string) (*AWSSQSConfig, error) {
	var as *AWSSQSConfig
	err := yaml.Unmarshal([]byte(config), &as)
	if err != nil {
		return nil, err
	}
	return as, nil
}
