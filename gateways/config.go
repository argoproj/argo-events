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
	"github.com/sirupsen/logrus"
	"os"

	"github.com/nats-io/go-nats"

	"github.com/argoproj/argo-events/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwclientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	snats "github.com/nats-io/go-nats-streaming"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GatewayConfig provides a generic event source for a gateway
type GatewayConfig struct {
	// Log provides fast and simple logger dedicated to JSON output
	Log *logrus.Logger
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
		Log: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelGatewayName: gw.Name,
				common.LabelNamespace:   gw.Namespace,
			}).Logger,
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

	switch gw.Spec.EventProtocol.Type {
	case pc.HTTP:
		gc.sensorHttpPort = gw.Spec.EventProtocol.Http.Port
	case pc.NATS:
		if gc.natsConn, err = nats.Connect(gw.Spec.EventProtocol.Nats.URL); err != nil {
			panic(fmt.Errorf("failed to obtain NATS standard connection. err: %+v", err))
		}
		gc.Log.WithField(common.LabelURL, gw.Spec.EventProtocol.Nats.URL).Info("connected to nats service")

		if gc.gw.Spec.EventProtocol.Nats.Type == pc.Streaming {
			gc.natsStreamingConn, err = snats.Connect(gc.gw.Spec.EventProtocol.Nats.ClusterId, gc.gw.Spec.EventProtocol.Nats.ClientId, snats.NatsConn(gc.natsConn))
			if err != nil {
				panic(fmt.Errorf("failed to obtain NATS streaming connection. err: %+v", err))
			}
			gc.Log.WithField(common.LabelURL, gw.Spec.EventProtocol.Nats.URL).Info("nats streaming connection successful")
		}
	}
	return gc
}
