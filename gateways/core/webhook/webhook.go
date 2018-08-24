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
	"github.com/argoproj/argo-events/common"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"github.com/google/go-cmp/cmp"
	"context"
	zlog "github.com/rs/zerolog"
	apiv1 "k8s.io/api/core/v1"
	"github.com/ghodss/yaml"
	"github.com/argoproj/argo-events/gateways"
	"log"
)

const (
	configName  = "webhook-gateway-configmap"
)

// hook is a general purpose REST API
type hook struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`

	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,2,opt,name=method"`
}

type webhook struct {
	gatewayConfig *gateways.GatewayConfig
	srv        *http.Server
	serverPort string
	registeredWebhooks []hook
}

// parses webhooks from gateway configuration
func (w *webhook) RunGateway(cm *apiv1.ConfigMap) (error) {
	if w.serverPort == "" {
		w.serverPort = cm.Data["port"]
		go func() {
			log.Println(fmt.Sprintf("server started listening on port %s", w.serverPort))
			log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%s", w.serverPort), nil))
		}()
	}
	// remove server port key
	delete(cm.Data, "port")
CheckAlreadyRegistered:
	for hookKey, hookValue := range cm.Data {
		var h *hook
		err := yaml.Unmarshal([]byte(hookValue), &h)
		if err != nil {
			return err
		}
		for _, registeredWebhook := range w.registeredWebhooks {
			if cmp.Equal(registeredWebhook, h) {
				w.gatewayConfig.Log.Warn().Interface("registered-webhook", registeredWebhook).Str("hook-name", hookKey).Msg("duplicate endpoint")
				goto CheckAlreadyRegistered
			}
		}
		w.registerWebhook(h)
		w.gatewayConfig.Log.Info().Str("hook-name", hookKey).Msg("webhook configured")
		w.registeredWebhooks = append(w.registeredWebhooks, *h)
	}
	return nil
}

// registers a http endpoint
func (w *webhook) registerWebhook(h *hook) {
	http.HandleFunc(h.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == h.Method {
			w.gatewayConfig.Log.Info().Msg("received a request, forwarding it to gateway transformer")
			http.Post(fmt.Sprintf("http://localhost:%s", w.gatewayConfig.TransformerPort), "application/octet-stream", request.Body)
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
		Log: zlog.New(os.Stdout).With().Logger(),
		Namespace: namespace,
		Clientset: clientset,
		TransformerPort: transformerPort,
	}
	w := &webhook{
		gatewayConfig: gatewayConfig,
		registeredWebhooks: []hook{},
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(w, context.Background(), configName)
	if err != nil {
		panic("failed to retrieve webhook configuration")
	}

	select {}
}
