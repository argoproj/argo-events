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
	"context"
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	eventsourceClientset "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	gwclientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
)

// GatewayContext holds the context for a gateway
type GatewayContext struct {
	// logger logs stuff
	logger *logrus.Logger
	// k8sClient is client for kubernetes API
	k8sClient kubernetes.Interface
	// eventSourceRef refers to event-source for the gateway
	eventSourceRef *v1alpha1.EventSourceRef
	// eventSourceClient is the client for EventSourceRef resource
	eventSourceClient eventsourceClientset.Interface
	// name of the gateway
	name string
	// namespace where gateway is deployed
	namespace string
	// gateway refers to Gateway custom resource
	gateway *v1alpha1.Gateway
	// gatewayClient is gateway clientset
	gatewayClient gwclientset.Interface
	// updated indicates whether gateway resource is updated
	updated bool
	// serverPort is gateway server port to listen events from
	serverPort string
	// eventSourceContexts stores information about current event sources that are running in the gateway
	eventSourceContexts map[string]*EventSourceContext
	// controllerInstanceId is instance ID of the gateway controller
	controllerInstanceID string
	// statusCh is used to communicate the status of an event source
	statusCh chan notification
	// EventBus config
	eventBusConfig *eventbusv1alpha1.BusConfig
	// EventBus subject
	eventBusSubject string
	// POD name
	podName string
}

// EventSourceContext contains information of a event source for gateway to run.
type EventSourceContext struct {
	// source holds the actual event source
	source *gateways.EventSource
	// ctx contains context for the connection
	ctx context.Context
	// cancel upon invocation cancels the connection context
	cancel context.CancelFunc
	// client is grpc client
	client gateways.EventingClient
	// conn is grpc connection
	conn *grpc.ClientConn
}

// NewGatewayContext returns a new gateway context
func NewGatewayContext() *GatewayContext {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	name, ok := os.LookupEnv(common.EnvVarResourceName)
	if !ok {
		panic("gateway name not provided")
	}
	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		panic("no namespace provided")
	}
	controllerInstanceID, ok := os.LookupEnv(common.EnvVarControllerInstanceID)
	if !ok {
		panic("gateway controller instance ID is not provided")
	}
	serverPort, ok := os.LookupEnv(common.EnvVarGatewayServerPort)
	if !ok {
		panic("server port is not provided")
	}

	busConfig := &eventbusv1alpha1.BusConfig{}
	encodedBusConfigSpec := os.Getenv(common.EnvVarEventBusConfig)
	if len(encodedBusConfigSpec) > 0 {
		busConfigSpec, err := base64.StdEncoding.DecodeString(encodedBusConfigSpec)
		if err != nil {
			panic(errors.Errorf("failed to decode bus config string. err: %+v", err))
		}
		if err = json.Unmarshal([]byte(busConfigSpec), busConfig); err != nil {
			panic(errors.Errorf("failed to unmarshal bus config object. err: %+v", err))
		}
	}

	ebSubject, defined := os.LookupEnv(common.EnvVarEventBusSubject)
	if !defined {
		panic(errors.Errorf("required environment variable '%s' not defined", common.EnvVarEventBusSubject))
	}

	podName, defined := os.LookupEnv("POD_NAME")
	if !defined {
		panic(errors.New("Environment POD_NAME is missing"))
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gatewayClient := gwclientset.NewForConfigOrDie(restConfig)
	eventSourceClient := eventsourceClientset.NewForConfigOrDie(restConfig)

	gateway, err := gatewayClient.ArgoprojV1alpha1().Gateways(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	gatewayConfig := &GatewayContext{
		logger: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelResourceName: gateway.Name,
				common.LabelNamespace:    gateway.Namespace,
			}).Logger,
		k8sClient:            clientset,
		namespace:            namespace,
		name:                 name,
		eventSourceContexts:  make(map[string]*EventSourceContext),
		eventSourceRef:       gateway.Spec.EventSourceRef,
		eventSourceClient:    eventSourceClient,
		gatewayClient:        gatewayClient,
		gateway:              gateway,
		controllerInstanceID: controllerInstanceID,
		serverPort:           serverPort,
		statusCh:             make(chan notification),
		eventBusConfig:       busConfig,
		eventBusSubject:      ebSubject,
		podName:              podName,
	}

	return gatewayConfig
}
