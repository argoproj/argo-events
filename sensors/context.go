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

	"github.com/Shopify/sarama"
	"github.com/apache/openwhisk-client-go/whisk"
	"github.com/aws/aws-sdk-go/service/lambda"
	natslib "github.com/nats-io/go-nats"
	"google.golang.org/grpc"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	sensormetrics "github.com/argoproj/argo-events/metrics"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// SensorContext contains execution context for Sensor
type SensorContext struct {
	// KubeClient is the kubernetes client
	KubeClient kubernetes.Interface
	// ClientPool manages a pool of dynamic clients.
	DynamicClient dynamic.Interface
	// Sensor object
	Sensor *v1alpha1.Sensor
	// EventBus config
	EventBusConfig *eventbusv1alpha1.BusConfig
	// EventBus subject
	EventBusSubject string
	// httpClients holds the reference to HTTP clients for HTTP triggers.
	httpClients map[string]*http.Client
	// customTriggerClients holds the references to the gRPC clients for the custom trigger servers
	customTriggerClients map[string]*grpc.ClientConn
	// http client to send slack messages.
	slackHTTPClient *http.Client
	// kafkaProducers holds references to the active kafka producers
	kafkaProducers map[string]sarama.AsyncProducer
	// natsConnections holds the references to the active nats connections.
	natsConnections map[string]*natslib.Conn
	// awsLambdaClients holds the references to active AWS Lambda clients.
	awsLambdaClients map[string]*lambda.Lambda
	// openwhiskClients holds the references to active OpenWhisk clients.
	openwhiskClients map[string]*whisk.Client

	metrics *sensormetrics.Metrics
}

// NewSensorContext returns a new sensor execution context.
func NewSensorContext(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, sensor *v1alpha1.Sensor, eventBusConfig *eventbusv1alpha1.BusConfig, eventBusSubject string, metrics *sensormetrics.Metrics) *SensorContext {
	return &SensorContext{
		KubeClient:           kubeClient,
		DynamicClient:        dynamicClient,
		Sensor:               sensor,
		EventBusConfig:       eventBusConfig,
		EventBusSubject:      eventBusSubject,
		httpClients:          make(map[string]*http.Client),
		customTriggerClients: make(map[string]*grpc.ClientConn),
		slackHTTPClient: &http.Client{
			Timeout: time.Minute * 5,
		},
		kafkaProducers:   make(map[string]sarama.AsyncProducer),
		natsConnections:  make(map[string]*natslib.Conn),
		awsLambdaClients: make(map[string]*lambda.Lambda),
		openwhiskClients: make(map[string]*whisk.Client),
		metrics:          metrics,
	}
}
