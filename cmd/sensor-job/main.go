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
	"fmt"
	"os"
	"os/exec"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/job"
	sensorclientset "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"github.com/hashicorp/go-plugin"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)

	config, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	sensorClientset := sensorclientset.NewForConfigOrDie(config)

	jobName, ok := os.LookupEnv(common.EnvVarJobName)
	if !ok {
		panic(fmt.Errorf("Unable to get job name from environment variable %s", common.EnvVarJobName))
	}
	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		panic(fmt.Errorf("Unable to get job namespace from environment variable %s", common.EnvVarNamespace))
	}

	sensor, err := sensorClientset.ArgoprojV1alpha1().Sensors(namespace).Get(common.ParseJobPrefix(jobName), metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	log := logger.With(zap.String("sensor", sensor.Name))

	// initialize the plugins
	pluginFile := os.Getenv("SIGNAL_PLUGIN")
	pluginClient := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: job.Handshake,
		Plugins:         job.PluginMap,
		Cmd:             exec.Command(pluginFile),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC,
		},
	})
	defer pluginClient.Kill()

	streamClient, err := pluginClient.Client()
	if err != nil {
		panic(err)
	}

	err = job.New(config, sensorClientset, log).Run(sensor.DeepCopy(), streamClient)
	if err != nil {
		panic(err.Error())
	}
}
