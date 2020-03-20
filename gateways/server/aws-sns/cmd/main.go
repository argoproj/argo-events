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
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/gateways/server/aws-sns"
	"k8s.io/client-go/kubernetes"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		panic("namespace is not provided")
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)

	server.StartGateway(&aws_sns.EventListener{
		Logger:    common.NewArgoEventsLogger(),
		K8sClient: clientset,
		Namespace: namespace,
	})
}
