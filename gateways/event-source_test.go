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
	"sync"
	"testing"

	"github.com/argoproj/argo-events/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwfake "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func getGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		Log:        common.NewArgoEventsLogger(),
		serverPort: "1234",
		StatusCh:   make(chan EventSourceStatus),
		gw: &v1alpha1.Gateway{
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
		Clientset: fake.NewSimpleClientset(),
		gwcs:      gwfake.NewSimpleClientset(),
	}
}

type testEventSourceExecutor struct{}

func (ese *testEventSourceExecutor) StartEventSource(eventSource *EventSource, eventStream Eventing_StartEventSourceServer) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
	_ = eventStream.Send(&Event{
		Name:    eventSource.Name,
		Payload: []byte("test payload"),
	})

	<-eventStream.Context().Done()

	return nil
}

func (ese *testEventSourceExecutor) ValidateEventSource(ctx context.Context, eventSource *EventSource) (*ValidEventSource, error) {
	return &ValidEventSource{
		IsValid: true,
	}, nil
}

func TestEventSources(t *testing.T) {
	_ = os.Setenv(common.EnvVarGatewayServerPort, "1234")
	go StartGateway(&testEventSourceExecutor{})
	gc := getGatewayConfig()

	var eventSrcCtxMap map[string]*EventSourceContext
	var eventSourceKeys []string

	convey.Convey("Given a gateway configmap, create event sources", t, func() {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gateway-configmap",
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				"event-source-1": `
testKey: testValue
`,
			},
		}
		fakeclientset := fake.NewSimpleClientset()
		_, err := fakeclientset.CoreV1().ConfigMaps(cm.Namespace).Create(cm)
		convey.So(err, convey.ShouldBeNil)

		eventSrcCtxMap, err = gc.createInternalEventSources(cm)
		convey.So(err, convey.ShouldBeNil)
		convey.So(eventSrcCtxMap, convey.ShouldNotBeNil)
		convey.So(len(eventSrcCtxMap), convey.ShouldEqual, 1)
		for _, data := range eventSrcCtxMap {
			convey.So(data.Data.Config, convey.ShouldEqual, `
testKey: testValue
`)
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
		for event := range gc.StatusCh {
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
