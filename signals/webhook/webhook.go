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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"os"
	"github.com/google/go-cmp/cmp"
	"context"
	zlog "github.com/rs/zerolog"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"github.com/ghodss/yaml"
)

// hook is a general purpose REST API
type webhookSignal struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`

	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,2,opt,name=method"`
}

type webhook struct {
	srv        *http.Server
	clientset  *kubernetes.Clientset
	config     string
	namespace  string
	log zlog.Logger
	serverPort string
	transformerPort string
	registeredWebhooks []webhookSignal
}

func (w *webhook) WatchGatewayTransformerConfigMap(ctx context.Context, name string) (cache.Controller, error) {
	source := w.newStoreConfigMapWatch(name)
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					w.log.Info().Str("config-map", name).Msg("detected ConfigMap update. Updating the controller config.")
					err := w.parseWebhooks(cm)
					if err != nil {
						w.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					w.log.Info().Msg("detected ConfigMap update. Updating the controller config.")
					err := w.parseWebhooks(newCm)
					if err != nil {
						w.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (w *webhook) newStoreConfigMapWatch(name string) *cache.ListWatch {
	x := w.clientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(w.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(w.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// parses webhooks from gateway configuration
func (w *webhook) parseWebhooks(cm *apiv1.ConfigMap) (error) {
	// remove server port key
	delete(cm.Data, "port")
CheckAlreadyRegistered:
	for hookKey, hookValue := range cm.Data {
		var h *webhookSignal
		err := yaml.Unmarshal([]byte(hookValue), &h)
		if err != nil {
			return err
		}
		for _, registeredWebhook := range w.registeredWebhooks {
			if cmp.Equal(registeredWebhook, h) {
				w.log.Warn().Interface("registered-webhook", registeredWebhook).Str("hook-name", hookKey).Msg("duplicate endpoint")
				goto CheckAlreadyRegistered
			}
		}
		w.registerWebhook(h)
		w.log.Info().Str("hook-name", hookKey).Msg("webhook configured")
		w.registeredWebhooks = append(w.registeredWebhooks, *h)
	}
	return nil
}

// registers a http endpoint
func (w *webhook) registerWebhook(h *webhookSignal) {
	http.HandleFunc(h.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == h.Method {
			w.log.Info().Msg("received a request, forwarding it to gateway transformer")
			http.Post(fmt.Sprintf("http://localhost:%s", w.transformerPort), "application/octet-stream", request.Body)
		} else {
			w.log.Warn().Str("expected", h.Method).Str("actual", request.Method).Msg("http method mismatch")
		}
	})
}

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	// Todo: hardcoded for now, move it to constants
	config := "webhook-gateway-configmap"

	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}

	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)

	w := &webhook{
		clientset:  clientset,
		config:     config,
		namespace:  namespace,
		log:        zlog.New(os.Stdout).With().Logger(),
		transformerPort: transformerPort,
	}

	configmap, err := w.clientset.CoreV1().ConfigMaps(w.namespace).Get(w.config, metav1.GetOptions{})
	if err != nil {
		panic(fmt.Errorf("failed to get webhook configuration. Err: %+v", err))
	}
	w.serverPort = configmap.Data["port"]
	_, err = w.WatchGatewayTransformerConfigMap(context.Background(), w.config)
	if err != nil {
		panic("failed to retrieve webhook configuration")
	}

	log.Println(fmt.Sprintf("server started listening on port %s", w.serverPort))
	log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%s", w.serverPort), nil))
}
