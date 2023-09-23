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

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	sensormetrics "github.com/argoproj/argo-events/metrics"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	openwhisk "github.com/argoproj/argo-events/sensors/triggers/apache-openwhisk"
	lambda "github.com/argoproj/argo-events/sensors/triggers/aws-lambda"
	azureeventhub "github.com/argoproj/argo-events/sensors/triggers/azure-event-hubs"
	azureservicebus "github.com/argoproj/argo-events/sensors/triggers/azure-service-bus"
	customtrigger "github.com/argoproj/argo-events/sensors/triggers/custom-trigger"
	httptrigger "github.com/argoproj/argo-events/sensors/triggers/http"
	"github.com/argoproj/argo-events/sensors/triggers/kafka"
	"github.com/argoproj/argo-events/sensors/triggers/nats"
	"github.com/argoproj/argo-events/sensors/triggers/pulsar"
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
	httpClients *httptrigger.HTTPClientMap
	// customTriggerClients holds the references to the gRPC clients for the custom trigger servers
	customTriggerClients *customtrigger.CustomTriggerClientMap
	// http client to send slack messages.
	slackHTTPClient *http.Client
	// kafkaProducers holds references to the active kafka producers
	kafkaProducers *kafka.AsyncProducerMap
	// pulsarProducers holds references to the active pulsar producers
	pulsarProducers *pulsar.AsyncProducerMap
	// natsConnections holds the references to the active nats connections.
	natsConnections *nats.NATSConnectionMap
	// awsLambdaClients holds the references to active AWS Lambda clients.
	awsLambdaClients *lambda.LambdaClientMap
	// openwhiskClients holds the references to active OpenWhisk clients.
	openwhiskClients *openwhisk.OpenWhiskClientMap
	// azureEventHubsClients holds the references to active Azure Event Hub clients.
	azureEventHubsClients *azureeventhub.EventhubClientMap
	// azureServiceBusClients holds the references to active Azure Service Bus clients.
	azureServiceBusClients *azureservicebus.ServicebusSenderMap
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
		httpClients:          httptrigger.NewHTTPClientMap(),
		customTriggerClients: customtrigger.NewCustomTriggerClientMap(),
		slackHTTPClient: &http.Client{
			Timeout: time.Minute * 5,
		},
		kafkaProducers:         kafka.NewAsyncProducerMap(),
		pulsarProducers:        pulsar.NewAsyncProducerMap(),
		natsConnections:        nats.NewNATSConnectionMap(),
		awsLambdaClients:       lambda.NewLambdaClientMap(),
		openwhiskClients:       openwhisk.NewOpenWhiskClientMap(),
		azureEventHubsClients:  azureeventhub.NewEventhubClientMap(),
		azureServiceBusClients: azureservicebus.NewServicebusSenderMap(),
		metrics:                metrics,
	}
}
