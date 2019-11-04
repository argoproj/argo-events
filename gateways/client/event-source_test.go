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
	"os"
	"sync"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	eventSourceV1Alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	eventSourceFake "github.com/argoproj/argo-events/pkg/client/eventsources/clientset/versioned/fake"
	gwfake "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func getGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		logger:     common.NewArgoEventsLogger(),
		serverPort: "1234",
		statusCh:   make(chan EventSourceStatus),
		gateway: &v1alpha1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-agteway",
				Namespace: "test-nm",
			},
			Spec: v1alpha1.GatewaySpec{
				Watchers: &v1alpha1.NotificationWatchers{
					Sensors: []v1alpha1.SensorNotificationWatcher{},
				},
				EventProtocol: &pc.EventProtocol{
					Type: pc.HTTP,
					Http: pc.Http{
						Port: "9000",
					},
				},
			},
		},
		k8sClient:     fake.NewSimpleClientset(),
		gatewayClient: gwfake.NewSimpleClientset(),
	}
}

type testEventSourceExecutor struct{}

func (ese *testEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
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

func (ese *testEventSourceExecutor) ValidateEventSource(ctx context.Context, eventSource *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return &gateways.ValidEventSource{
		IsValid: true,
	}, nil
}

func TestEventSources(t *testing.T) {
	_ = os.Setenv(common.EnvVarGatewayServerPort, "1234")
	go server.StartGateway(&testEventSourceExecutor{})
	gc := getGatewayConfig()

	var eventSrcCtxMap map[string]*EventSourceContext
	var eventSourceKeys []string

	convey.Convey("Given a EventSource resource, create internal event sources", t, func() {
		eventSource := &eventSourceV1Alpha1.EventSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-event-source",
				Namespace: "test-namespace",
			},
			Spec: &eventSourceV1Alpha1.EventSourceSpec{
				Webhook: map[string]webhook.Context{
					"fake": {
						Port:     "80",
						URL:      "fake-url",
						Endpoint: "xx",
						Method:   "GET",
					},
				},
			},
		}

		fakeclientset := eventSourceFake.NewSimpleClientset()
		_, err := fakeclientset.ArgoprojV1alpha1().EventSources(eventSource.Namespace).Create(eventSource)
		convey.So(err, convey.ShouldBeNil)

		eventSrcCtxMap, err = gc.createInternalEventSources(eventSource)
		convey.So(err, convey.ShouldBeNil)
		convey.So(eventSrcCtxMap, convey.ShouldNotBeNil)
		convey.So(len(eventSrcCtxMap), convey.ShouldEqual, 1)
		for _, data := range eventSrcCtxMap {
			convey.So(data.source.Value, convey.ShouldEqual, `
testKey: testValue
`)
			convey.So(data.source.Version, convey.ShouldEqual, "v0.10")
		}
	})

	convey.Convey("Given old and new event sources, return diff", t, func() {
		gc.registeredConfigs = make(map[string]*EventSourceContext)
		staleEventSources, newEventSources := gc.diffEventSources(eventSrcCtxMap)
		convey.So(staleEventSources, convey.ShouldBeEmpty)
		convey.So(newEventSources, convey.ShouldNotBeEmpty)
		convey.So(len(newEventSources), convey.ShouldEqual, 1)
		eventSourceKeys = newEventSources
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		i := 0
		for event := range gc.statusCh {
			switch event.Phase {
			case v1alpha1.NodePhaseRunning:
				convey.Convey("Event source is running", t, func() {
					convey.So(i, convey.ShouldEqual, 0)
					convey.So(event.Message, convey.ShouldEqual, "event_source_is_running")
					i++
					go gc.stopEventSources(eventSourceKeys)
				})
			case v1alpha1.NodePhaseError:
				convey.Convey("Event source is in error", t, func() {
					convey.So(i, convey.ShouldNotEqual, 0)
					convey.So(event.Message, convey.ShouldEqual, "failed_to_receive_event_from_event_source_stream")
				})

			case v1alpha1.NodePhaseRemove:
				convey.Convey("Event source should be removed", t, func() {
					convey.So(i, convey.ShouldNotEqual, 0)
					convey.So(event.Message, convey.ShouldEqual, "event_source_is_removed")
				})
				goto end
			}
		}
	end:
		wg.Done()
	}()

	convey.Convey("Given new event sources, start consuming events", t, func() {
		gc.startEventSources(eventSrcCtxMap, eventSourceKeys)
		wg.Wait()
	})
}
