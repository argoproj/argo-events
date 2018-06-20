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

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/job"
	"github.com/argoproj/argo-events/job/amqp"
	"github.com/argoproj/argo-events/job/artifact"
	"github.com/argoproj/argo-events/job/calendar"
	"github.com/argoproj/argo-events/job/kafka"
	"github.com/argoproj/argo-events/job/mqtt"
	"github.com/argoproj/argo-events/job/nats"
	"github.com/argoproj/argo-events/job/resource"
	"github.com/argoproj/argo-events/job/webhook"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorclientset "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)

	config, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err.Error())
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
		panic(err.Error())
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err.Error())
	}
	sensorLogger := logger.With(zap.String("sensor", sensor.Name))

	// find which signals to run
	registers, err := getSignalRegisters(sensor.Spec.Signals)
	if err != nil {
		panic(err.Error())
	}

	err = job.New(config, sensorClientset, sensorLogger).Run(sensor.DeepCopy(), registers)
	if err != nil {
		panic(err.Error())
	}
}

func getSignalRegisters(signals []v1alpha1.Signal) ([]func(*job.ExecutorSession), error) {
	var registerFuncs []func(*job.ExecutorSession)
	for _, signal := range signals {
		switch signal.GetType() {
		case v1alpha1.SignalTypeStream:
			streamFactory, err := resolveSignalStreamFactory(*signal.Stream)
			if err != nil {
				return registerFuncs, err
			}
			registerFuncs = append(registerFuncs, streamFactory)
		case v1alpha1.SignalTypeArtifact:
			registerFuncs = append(registerFuncs, artifact.Artifact)
			// for artifacts, need to find which stream to use
			streamFactory, err := resolveSignalStreamFactory(signal.Artifact.NotificationStream)
			if err != nil {
				return registerFuncs, err
			}
			registerFuncs = append(registerFuncs, streamFactory)
		case v1alpha1.SignalTypeResource:
			registerFuncs = append(registerFuncs, resource.Resource)
		case v1alpha1.SignalTypeCalendar:
			registerFuncs = append(registerFuncs, calendar.Calendar)
		case v1alpha1.SignalTypeWebhook:
			registerFuncs = append(registerFuncs, webhook.Webhook)
		default:
			return registerFuncs, fmt.Errorf("%s signal type not supported", signal.GetType())
		}
	}
	return registerFuncs, nil
}

func resolveSignalStreamFactory(stream v1alpha1.Stream) (func(*job.ExecutorSession), error) {
	switch stream.Type {
	case nats.StreamTypeNats:
		return nats.NATS, nil
	case mqtt.StreamTypeMqtt:
		return mqtt.MQTT, nil
	case amqp.StreamTypeAMQP:
		return amqp.AMQP, nil
	case kafka.StreamTypeKafka:
		return kafka.Kafka, nil
	default:
		return nil, fmt.Errorf("unsupported stream type")
	}
}
