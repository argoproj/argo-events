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
	"github.com/argoproj/argo-events/sdk"
	"github.com/argoproj/argo-events/signals/resource"
	"github.com/micro/go-micro"
	k8s "github.com/micro/kubernetes/go/micro"
)

func main() {
	svc := k8s.NewService(micro.Name("artifact"), micro.Metadata(sdk.SignalMetadata))
	svc.Init()

	// kubernetes configuration
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	rest, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	sdk.RegisterSignalServiceHandler(svc.Server(), sdk.NewMicroSignalServer(resource.New(rest)))

	if err := svc.Run(); err != nil {
		panic(err)
	}
}
