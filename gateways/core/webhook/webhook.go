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
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
	hs "github.com/mitchellh/hashstructure"
	zlog "github.com/rs/zerolog"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"os"
	"io/ioutil"
	"bytes"
)

const (
	configName = "webhook-gateway-configmap"
)

// hook is a general purpose REST API
type hook struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`

	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,2,opt,name=method"`
}

// webhook contains gateway configuration and registered endpoints
type webhook struct {
	// gatewayConfig contains general configuration for gateway
	gatewayConfig *gateways.GatewayConfig
	// srv is reference to http server
	srv *http.Server
	// serverPort is port on which server is listening
	serverPort string
	// registeredWebhooks contains map of registered http endpoints
	registeredWebhooks map[uint64]*hook
}

// parses webhooks from gateway configuration
func (w *webhook) RunGateway(cm *apiv1.ConfigMap) error {
	if w.serverPort == "" {
		w.serverPort = cm.Data["port"]
		go func() {
			log.Println(fmt.Sprintf("server started listening on port %s", w.serverPort))
			log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%s", w.serverPort), nil))
		}()
	}
	// remove server port key, if already removed its a no-op
	delete(cm.Data, "port")
	for hookConfigKey, hookConfigValue := range cm.Data {
		var h *hook
		err := yaml.Unmarshal([]byte(hookConfigValue), &h)
		if err != nil {
			return err
		}
		key, err := hs.Hash(h, &hs.HashOptions{})
		if err != nil {
			w.gatewayConfig.Log.Warn().Err(err).Msg("failed to get hash of configuration")
			continue
		}

		if _, ok := w.registeredWebhooks[key]; ok {
			w.gatewayConfig.Log.Warn().Interface("config", h).Msg("duplicate configuration")
			continue
		}
		w.registeredWebhooks[key] = h
		w.registerWebhook(h, hookConfigKey)
		w.gatewayConfig.Log.Info().Str("hook-name", hookConfigKey).Msg("configured")
	}
	return nil
}

// registers a http endpoint
func (w *webhook) registerWebhook(h *hook, source string) {
	http.HandleFunc(h.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
		w.gatewayConfig.Log.Info().Msg("received a request. wtf!!!")
		if request.Method == h.Method {
			body, err := ioutil.ReadAll(request.Body)
			if err != nil {
				w.gatewayConfig.Log.Panic().Err(err).Msg("failed to parse request body")
			}

			payload, err := gateways.CreateTransformPayload(body, source)
			if err != nil {
				w.gatewayConfig.Log.Panic().Err(err).Msg("failed to transform request body")
			}

			w.gatewayConfig.Log.Info().Msg("dispatching the event to gateway-transformer...")
			_, err = http.Post(fmt.Sprintf("http://localhost:%s", w.gatewayConfig.TransformerPort), "application/octet-stream", bytes.NewReader(payload))
			if err != nil {
				w.gatewayConfig.Log.Warn().Err(err).Msg("failed to dispatch the event.")
			}
		} else {
			w.gatewayConfig.Log.Warn().Str("expected", h.Method).Str("actual", request.Method).Msg("http method mismatch")
		}
	})
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
		Log:             zlog.New(os.Stdout).With().Logger(),
		Namespace:       namespace,
		Clientset:       clientset,
		TransformerPort: transformerPort,
	}
	w := &webhook{
		gatewayConfig:      gatewayConfig,
		registeredWebhooks: make(map[uint64]*hook),
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(w, context.Background(), configName)
	if err != nil {
		panic("failed to retrieve webhook configuration")
	}

	select {}
}
