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

package gateways

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/argo-events/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	eventsourceClientset "github.com/argoproj/argo-events/pkg/client/eventsources/clientset/versioned"
	gwclientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"github.com/nats-io/go-nats"
	snats "github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GatewayConfig provides a generic event source for a gateway
type GatewayConfig struct {
	// Logger logs stuff
	Logger *logrus.Logger
	// K8sClient is client for kubernetes API
	K8sClient kubernetes.Interface
	// EventSourceClient is the client for EventSourceRef resource
	EventSourceClient eventsourceClientset.Interface
	// Name of the gateway
	Name string
	// Namespace where gateway is deployed
	Namespace string
	// KubeConfig is the rest client config
	KubeConfig *rest.Config
	// gateway refers to Gateway custom resource
	gateway *v1alpha1.Gateway
	// gatewayClient is gateway clientset
	gatewayClient gwclientset.Interface
	// updated indicates whether gateway resource is updated
	updated bool
	// serverPort is gateway server port to listen events from
	serverPort string
	// registeredConfigs stores information about current event sources that are running in the gateway
	registeredConfigs map[string]*EventSourceContext
	// eventSourceRef refers to event-source for the gateway
	eventSourceRef *v1alpha1.EventSourceRef
	// controllerInstanceId is instance ID of the gateway controller
	controllerInstanceID string
	// StatusCh is used to communicate the status of an event source
	StatusCh chan EventSourceStatus
	// natsConn is the standard nats connection used to publish events to cluster. Only used if dispatch protocol is NATS
	natsConn *nats.Conn
	// natsStreamingConn is the nats connection used for streaming.
	natsStreamingConn snats.Conn
	// sensorHttpPort is the http server running in sensor that listens to event. Only used if dispatch protocol is HTTP
	sensorHttpPort string
}

// EventSourceContext contains information of a event source for gateway to run.
type EventSourceContext struct {
	// Source holds the actual event source
	Source *EventSource
	// Ctx contains context for the connection
	Ctx context.Context
	// Cancel upon invocation cancels the connection context
	Cancel context.CancelFunc
	// Client is grpc client
	Client EventingClient
	// Conn is grpc connection
	Conn *grpc.ClientConn
}

// GatewayEvent is the internal representation of an event.
type GatewayEvent struct {
	// Src is source of event
	Src string `json:"src"`
	// Payload contains event data
	Payload []byte `json:"payload"`
}

// NewGatewayConfiguration returns a new gateway configuration
func NewGatewayConfiguration() *GatewayConfig {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	name, ok := os.LookupEnv(common.EnvVarGatewayName)
	if !ok {
		panic("gateway name not provided")
	}
	namespace, ok := os.LookupEnv(common.EnvVarGatewayNamespace)
	if !ok {
		panic("no namespace provided")
	}
	controllerInstanceID, ok := os.LookupEnv(common.EnvVarGatewayControllerInstanceID)
	if !ok {
		panic("gateway controller instance ID is not provided")
	}
	serverPort, ok := os.LookupEnv(common.EnvVarGatewayServerPort)
	if !ok {
		panic("server port is not provided")
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gatewayClient := gwclientset.NewForConfigOrDie(restConfig)
	gateway, err := gatewayClient.ArgoprojV1alpha1().Gateways(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	gatewayConfig := &GatewayConfig{
		Logger: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelGatewayName: gateway.Name,
				common.LabelNamespace:   gateway.Namespace,
			}).Logger,
		K8sClient:            clientset,
		Namespace:            namespace,
		Name:                 name,
		KubeConfig:           restConfig,
		registeredConfigs:    make(map[string]*EventSourceContext),
		eventSourceRef:       &gateway.Spec.EventSourceRef,
		gatewayClient:        gatewayClient,
		gateway:              gateway,
		controllerInstanceID: controllerInstanceID,
		serverPort:           serverPort,
		StatusCh:             make(chan EventSourceStatus),
	}

	switch gateway.Spec.EventProtocol.Type {
	case pc.HTTP:
		gatewayConfig.sensorHttpPort = gateway.Spec.EventProtocol.Http.Port
	case pc.NATS:
		if gatewayConfig.natsConn, err = nats.Connect(gateway.Spec.EventProtocol.Nats.URL); err != nil {
			panic(fmt.Errorf("failed to obtain NATS standard connection. err: %+v", err))
		}
		gatewayConfig.Logger.WithField(common.LabelURL, gateway.Spec.EventProtocol.Nats.URL).Infoln("connected to nats service")

		if gatewayConfig.gateway.Spec.EventProtocol.Nats.Type == pc.Streaming {
			gatewayConfig.natsStreamingConn, err = snats.Connect(gatewayConfig.gateway.Spec.EventProtocol.Nats.ClusterId, gatewayConfig.gateway.Spec.EventProtocol.Nats.ClientId, snats.NatsConn(gatewayConfig.natsConn))
			if err != nil {
				panic(fmt.Errorf("failed to obtain NATS streaming connection. err: %+v", err))
			}
			gatewayConfig.Logger.WithField(common.LabelURL, gateway.Spec.EventProtocol.Nats.URL).Infoln("nats streaming connection successful")
		}
	}
	return gatewayConfig
}
