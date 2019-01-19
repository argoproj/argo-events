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
	"github.com/nats-io/go-nats"
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwclientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GatewayConfig provides a generic event source for a gateway
type GatewayConfig struct {
	// Log provides fast and simple logger dedicated to JSON output
	Log zerolog.Logger
	// Clientset is client for kubernetes API
	Clientset kubernetes.Interface
	// Name is gateway name
	Name string
	// Namespace is namespace for the gateway to run inside
	Namespace string
	// KubeConfig rest client config
	KubeConfig *rest.Config
	// gateway holds Gateway custom resource
	gw *v1alpha1.Gateway
	// gwClientset is gateway clientset
	gwcs gwclientset.Interface
	// updated indicates whether gateway resource is updated
	updated bool
	// serverPort is gateway server port to listen events from
	serverPort string
	// registeredConfigs stores information about current event sources that are running in the gateway
	registeredConfigs map[string]*EventSourceContext
	// configName is name of configmap that contains run event source/s for the gateway
	configName string
	// controllerInstanceId is instance ID of the gateway controller
	controllerInstanceID string
	// StatusCh is used to communicate the status of an event source
	StatusCh chan EventSourceStatus
	// natsConn is the nast client used to publish events to cluster. Only used if dispatch protocol is NATS
	natsConn *nats.Conn
	// sensorHttpPort is the http server running in sensor that listens to event. Only used if dispatch protocol is HTTP
	sensorHttpPort string
}

// EventSourceContext contains information of a event source for gateway to run.
type EventSourceContext struct {
	// Data holds the actual event source
	Data *EventSourceData
	// Ctx contains context for the connection
	Ctx context.Context
	// Cancel upon invocation cancels the connection context
	Cancel context.CancelFunc
	// Client is grpc client
	Client EventingClient
	// Conn is grpc connection
	Conn *grpc.ClientConn
}

// EventSourceData holds the actual event source
type EventSourceData struct {
	// Unique ID for event source
	ID string `json:"id"`
	// Src contains name of the event source
	Src string `json:"src"`
	// Config contains the event source
	Config string `json:"config"`
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
	configName, ok := os.LookupEnv(common.EnvVarGatewayEventSourceConfigMap)
	if !ok {
		panic("gateway processor configmap is not provided")
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
	gwcs := gwclientset.NewForConfigOrDie(restConfig)
	gw, err := gwcs.ArgoprojV1alpha1().Gateways(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	gc := &GatewayConfig{
		Log:                  common.GetLoggerContext(common.LoggerConf()).Str("gateway-name", name).Str("gateway-namespace", namespace).Logger(),
		Clientset:            clientset,
		Namespace:            namespace,
		Name:                 name,
		KubeConfig:           restConfig,
		registeredConfigs:    make(map[string]*EventSourceContext),
		configName:           configName,
		gwcs:                 gwcs,
		gw:                   gw,
		controllerInstanceID: controllerInstanceID,
		serverPort:           serverPort,
		StatusCh:             make(chan EventSourceStatus),
	}

	switch gw.Spec.DispatchProtocol.Type {
	case v1alpha1.HTTPGateway:
		gc.sensorHttpPort = gw.Spec.DispatchProtocol.Http.Port
	case v1alpha1.NATSGateway:
		if gc.natsConn, err = nats.Connect(gw.Spec.DispatchProtocol.Nats.URL); err != nil {
			panic(fmt.Errorf("failed to connect to NATS cluster. err: %+v", err))
		}
	}
	return gc
}
