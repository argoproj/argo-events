/*
Copyright 2018 KompiTech GmbH

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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/community/github"
	"k8s.io/client-go/kubernetes"
	"os"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)
	namespace, ok := os.LookupEnv(common.EnvVarGatewayNamespace)
	if !ok {
		panic("namespace is not provided")
	}
	gateways.StartGateway(&github.GithubEventSourceExecutor{
		Log:       common.NewArgoEventsLogger(),
		Namespace: namespace,
		Clientset: clientset,
	})
}
