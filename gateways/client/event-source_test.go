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
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	esv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwfake "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func getGatewayContext() *GatewayContext {
	return &GatewayContext{
		logger:     common.NewArgoEventsLogger(),
		serverPort: "20000",
		statusCh:   make(chan notification),
		gateway: &v1alpha1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-gateway",
				Namespace: "fake-namespace",
			},
			Spec: v1alpha1.GatewaySpec{
				Subscribers: &v1alpha1.Subscribers{
					HTTP: []string{},
				},
				Type: apicommon.WebhookEvent,
			},
		},
		eventSourceContexts: make(map[string]*EventSourceContext),
		k8sClient:           fake.NewSimpleClientset(),
		gatewayClient:       gwfake.NewSimpleClientset(),
	}
}

func getEventSource() *esv1alpha1.EventSource {
	return &esv1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-event-source",
			Namespace: "fake-namespace",
		},
		Spec: &esv1alpha1.EventSourceSpec{
			Webhook: map[string]webhook.Context{
				"first-webhook": {
					Endpoint: "/first-webhook",
					Method:   http.MethodPost,
					Port:     "13000",
				},
			},
			Type: apicommon.WebhookEvent,
		},
	}
}

// Set up a fake gateway server
type testEventListener struct{}

func (listener *testEventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
	_ = eventStream.Send(&gateways.Event{
		Name:    eventSource.Name,
		Payload: []byte("test payload"),
	})

	<-eventStream.Context().Done()

	return nil
}

func (listener *testEventListener) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func getGatewayServer() *grpc.Server {
	srv := grpc.NewServer()
	gateways.RegisterEventingServer(srv, &testEventListener{})
	return srv
}

func TestInitEventSourceContexts(t *testing.T) {
	gatewayContext := getGatewayContext()
	eventSource := getEventSource().DeepCopy()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", gatewayContext.serverPort))
	if err != nil {
		panic(err)
	}

	server := getGatewayServer()
	stopCh := make(chan struct{})

	go func() {
		if err := server.Serve(lis); err != nil {
			return
		}
	}()

	go func() {
		<-stopCh
		server.GracefulStop()
		fmt.Println("server is stopped")
	}()

	contexts, err := gatewayContext.initEventSourceContexts(eventSource)
	assert.NoError(t, err)
	assert.NotNil(t, contexts)
	for _, esContext := range contexts {
		assert.Equal(t, "first-webhook", esContext.source.Name)
		assert.NotNil(t, esContext.conn)
	}

	stopCh <- struct{}{}

	time.Sleep(5 * time.Second)
}

func TestSyncEventSources(t *testing.T) {
	gatewayContext := getGatewayContext()
	eventSource := getEventSource().DeepCopy()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", gatewayContext.serverPort))
	if err != nil {
		panic(err)
	}

	server := getGatewayServer()
	stopCh := make(chan struct{})
	stopStatus := make(chan struct{})

	go func() {
		if err := server.Serve(lis); err != nil {
			fmt.Println(err)
			return
		}
	}()

	go func() {
		for {
			select {
			case <-gatewayContext.statusCh:
			case <-stopStatus:
				return
			}
		}
	}()

	go func() {
		<-stopCh
		server.GracefulStop()
		fmt.Println("server is stopped")
		stopStatus <- struct{}{}
	}()

	err = gatewayContext.syncEventSources(eventSource)
	assert.Nil(t, err)

	time.Sleep(5 * time.Second)

	delete(eventSource.Spec.Webhook, "first-webhook")

	eventSource.Spec.Webhook["second-webhook"] = webhook.Context{
		Endpoint: "/second-webhook",
		Method:   http.MethodPost,
		Port:     "13000",
	}

	err = gatewayContext.syncEventSources(eventSource)
	assert.Nil(t, err)

	time.Sleep(5 * time.Second)

	delete(eventSource.Spec.Webhook, "second-webhook")

	err = gatewayContext.syncEventSources(eventSource)
	assert.Nil(t, err)

	time.Sleep(5 * time.Second)

	stopCh <- struct{}{}

	time.Sleep(5 * time.Second)
}

func TestDiffEventSources(t *testing.T) {
	gatewayContext := getGatewayContext()
	eventSourceContexts := map[string]*EventSourceContext{
		"first-webhook": {},
	}
	assert.NotNil(t, eventSourceContexts)
	staleEventSources, newEventSources := gatewayContext.diffEventSources(eventSourceContexts)
	assert.Nil(t, staleEventSources)
	assert.NotNil(t, newEventSources)
	gatewayContext.eventSourceContexts = map[string]*EventSourceContext{
		"first-webhook": {},
	}
	delete(eventSourceContexts, "first-webhook")
	staleEventSources, newEventSources = gatewayContext.diffEventSources(eventSourceContexts)
	assert.NotNil(t, staleEventSources)
	assert.Nil(t, newEventSources)
}
