package sensors

/*
Copyright 2020 BlackRock, Inc.

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

import (
	"net/http"
	"time"

	eventhubs "github.com/Azure/azure-event-hubs-go/v3"
	servicebus "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/IBM/sarama"
	"github.com/apache/openwhisk-client-go/whisk"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/aws/aws-sdk-go/service/lambda"
	natslib "github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
	sensormetrics "github.com/argoproj/argo-events/metrics"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// SensorContext contains execution context for Sensor
type SensorContext struct {
	// kubeClient is the kubernetes client
	kubeClient kubernetes.Interface
	// dynamic clients.
	dynamicClient dynamic.Interface
	// Sensor object
	sensor *v1alpha1.Sensor
	// EventBus config
	eventBusConfig *eventbusv1alpha1.BusConfig
	// EventBus subject
	eventBusSubject string
	hostname        string

	// httpClients holds the reference to HTTP clients for HTTP triggers.
	httpClients common.StringKeyedMap[*http.Client]
	// customTriggerClients holds the references to the gRPC clients for the custom trigger servers
	customTriggerClients common.StringKeyedMap[*grpc.ClientConn]
	// http client to send slack messages.
	slackHTTPClient *http.Client
	// kafkaProducers holds references to the active kafka producers
	kafkaProducers common.StringKeyedMap[sarama.AsyncProducer]
	// pulsarProducers holds references to the active pulsar producers
	pulsarProducers common.StringKeyedMap[pulsar.Producer]
	// natsConnections holds the references to the active nats connections.
	natsConnections common.StringKeyedMap[*natslib.Conn]
	// awsLambdaClients holds the references to active AWS Lambda clients.
	awsLambdaClients common.StringKeyedMap[*lambda.Lambda]
	// openwhiskClients holds the references to active OpenWhisk clients.
	openwhiskClients common.StringKeyedMap[*whisk.Client]
	// azureEventHubsClients holds the references to active Azure Event Hub clients.
	azureEventHubsClients common.StringKeyedMap[*eventhubs.Hub]
	// azureServiceBusClients holds the references to active Azure Service Bus clients.
	azureServiceBusClients common.StringKeyedMap[*servicebus.Sender]
	metrics                *sensormetrics.Metrics
}

// NewSensorContext returns a new sensor execution context.
func NewSensorContext(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, sensor *v1alpha1.Sensor, eventBusConfig *eventbusv1alpha1.BusConfig, eventBusSubject, hostname string, metrics *sensormetrics.Metrics) *SensorContext {
	return &SensorContext{
		kubeClient:           kubeClient,
		dynamicClient:        dynamicClient,
		sensor:               sensor,
		eventBusConfig:       eventBusConfig,
		eventBusSubject:      eventBusSubject,
		hostname:             hostname,
		httpClients:          common.NewStringKeyedMap[*http.Client](),
		customTriggerClients: common.NewStringKeyedMap[*grpc.ClientConn](),
		slackHTTPClient: &http.Client{
			Timeout: time.Minute * 5,
		},
		kafkaProducers:         common.NewStringKeyedMap[sarama.AsyncProducer](),
		pulsarProducers:        common.NewStringKeyedMap[pulsar.Producer](),
		natsConnections:        common.NewStringKeyedMap[*natslib.Conn](),
		awsLambdaClients:       common.NewStringKeyedMap[*lambda.Lambda](),
		openwhiskClients:       common.NewStringKeyedMap[*whisk.Client](),
		azureEventHubsClients:  common.NewStringKeyedMap[*eventhubs.Hub](),
		azureServiceBusClients: common.NewStringKeyedMap[*servicebus.Sender](),
		metrics:                metrics,
	}
}
