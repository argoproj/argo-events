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
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const ArgoEventsEventSourceVersion = "v0.10"

// SQSEventSourceExecutor implements Eventing
type SQSEventSourceExecutor struct {
	Log *logrus.Logger
	// Clientset is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// sqsEventSource contains information to listen to AWS SQS
type sqsEventSource struct {
	// AccessKey refers K8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector `json:"accessKey"`

	// SecretKey refers K8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector `json:"secretKey"`

	// Region is AWS region
	Region string `json:"region"`

	// Queue is AWS SQS queue to listen to for messages
	Queue string `json:"queue"`

	// WaitTimeSeconds is The duration (in seconds) for which the call waits for a message to arrive
	// in the queue before returning.
	WaitTimeSeconds int64 `json:"waitTimeSeconds"`
}

func parseEventSource(es string) (interface{}, error) {
	var n *sqsEventSource
	err := yaml.Unmarshal([]byte(es), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
