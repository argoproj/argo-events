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
	natsio "github.com/nats-io/go-nats"
	"k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/api/core/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/argoproj/argo-events/gateways/core/stream"
	"github.com/ghodss/yaml"
	"net/http"
	"bytes"
	"os"
	"github.com/argoproj/argo-events/common"
	zlog "github.com/rs/zerolog"
	"context"
	"strings"
	"github.com/argoproj/argo-events/gateways"
)

const (
	subjectKey = "subject"
	configName = "nats-gateway-configmap"
)

// Contains configuration for nats gateway
type nats struct{
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig *gateways.GatewayConfig
	// registeredNATS is list of registered stream configurations
	registeredNATS []stream.Stream
}

// listens to messages published to subject of interest
func (n *nats) listen(s *stream.Stream) {
	conn, err := natsio.Connect(s.URL)
	if err != nil {
		n.gatewayConfig.Log.Error().Str("url", s.URL).Err(err).Msg("connection failed")
		return
	}
	n.gatewayConfig.Log.Info().Str("url", s.URL).Str("subject", s.Attributes[subjectKey]).Msg("connected")
	n.gatewayConfig.Log.Info().Str("server id", conn.ConnectedServerId()).Str("connected url", conn.ConnectedUrl()).
		Str("servers", strings.Join(conn.DiscoveredServers(), ",")).Msg("nats connection")
	_, err = conn.Subscribe(s.Attributes[subjectKey], func(msg *natsio.Msg) {
		n.gatewayConfig.Log.Info().Msg("received a msg, forwarding it to gateway transformer")
		http.Post(fmt.Sprintf("http://localhost:%s", n.gatewayConfig.TransformerPort), "application/octet-stream", bytes.NewReader(msg.Data))
	})
	if err != nil {
		n.gatewayConfig.Log.Error().Str("url", s.URL).Str("subject", s.Attributes[subjectKey]).Err(err).Msg("failed to subscribe to subject")
		return
	}
}

func (n *nats) RunGateway(cm *apiv1.ConfigMap) error {
CheckAlreadyRegistered:
	for nkey, nval := range cm.Data {
		var s *stream.Stream
		err := yaml.Unmarshal([]byte(nval), &s)
		if err != nil {
			n.gatewayConfig.Log.Error().Str("artifact", nkey).Err(err).Msg("failed to parse calendar schedule")
			return err
		}
		n.gatewayConfig.Log.Info().Interface("stream", *s).Msg("stream configuration")
		for _, rnats := range n.registeredNATS {
			if cmp.Equal(rnats, s) {
				n.gatewayConfig.Log.Warn().Str("url", s.URL).Str("subject-key", s.Attributes[subjectKey]).Msg("duplicate configuration")
				goto CheckAlreadyRegistered
			}
		}
		n.registeredNATS = append(n.registeredNATS, *s)
		go n.listen(s)
	}
	return nil
}

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}
	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)
	gatewayConfig := &gateways.GatewayConfig{
		Log: zlog.New(os.Stdout).With().Logger(),
		Namespace: namespace,
		Clientset: clientset,
		TransformerPort: transformerPort,
	}
	n := &nats{
		gatewayConfig: gatewayConfig,
		registeredNATS: []stream.Stream{},
	}

	_, err = gatewayConfig.WatchGatewayConfigMap(n, context.Background(), configName)
	if err != nil {
		n.gatewayConfig.Log.Error().Err(err).Msg("failed to update nats gateway confimap")
	}
	select {}
}
