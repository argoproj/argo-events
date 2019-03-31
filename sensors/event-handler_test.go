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

package sensors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-events/common"
	sensor2 "github.com/argoproj/argo-events/controllers/sensor"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorFake "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	discoveryFake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

var sensorStr = `
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: test-sensor
  labels:
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  template:
    containers:
      - name: "sensor"
        image: "argoproj/sensor"
        imagePullPolicy: Always
    serviceAccountName: argo-events-sa
  dependencies:
    - name: "test-gateway:test"
  eventProtocol:
    type: "HTTP"
    http:
      port: "9300"
  triggers:
    - template: 
        name: test-workflow-trigger
        group: argoproj.io
        version: v1alpha1
        kind: Workflow
        source:
          inline: |
            apiVersion: argoproj.io/v1alpha1
            kind: Workflow
            metadata:
              generateName: hello-world-
            spec:
              entrypoint: whalesay
              templates:
                - name: whalesay
                  container:
                    args:
                      - "hello world"
                    command:
                      - cowsay
                    image: "docker/whalesay:latest"
`

var podResourceList = metav1.APIResourceList{
	GroupVersion: metav1.GroupVersion{Group: "", Version: "v1"}.String(),
	APIResources: []metav1.APIResource{
		{Kind: "Pod", Namespaced: true, Name: "pods", SingularName: "pod", Group: "", Version: "v1", Verbs: []string{"create", "get"}},
	},
}

func getSensor() (*v1alpha1.Sensor, error) {
	var sensor v1alpha1.Sensor
	err := yaml.Unmarshal([]byte(sensorStr), &sensor)
	return &sensor, err
}

type mockHttpWriter struct {
	Status  int
	Payload []byte
}

func (m *mockHttpWriter) Header() http.Header {
	return http.Header{}
}

func (m *mockHttpWriter) Write(p []byte) (int, error) {
	m.Payload = p
	return 0, nil
}

func (m *mockHttpWriter) WriteHeader(statusCode int) {
	m.Status = statusCode
}

func getsensorExecutionCtx(sensor *v1alpha1.Sensor) *sensorExecutionCtx {
	kubeClientset := fake.NewSimpleClientset()
	fakeDiscoveryClient := kubeClientset.Discovery().(*discoveryFake.FakeDiscovery)
	clientPool := &FakeClientPool{
		Fake: kubeClientset.Fake,
	}
	fakeDiscoveryClient.Resources = append(fakeDiscoveryClient.Resources, &podResourceList)
	return &sensorExecutionCtx{
		kubeClient:           kubeClientset,
		discoveryClient:      fakeDiscoveryClient,
		clientPool:           clientPool,
		log:                  common.NewArgoEventsLogger(),
		sensorClient:         sensorFake.NewSimpleClientset(),
		sensor:               sensor,
		controllerInstanceID: "test-1",
		queue:                make(chan *updateNotification),
	}
}

func getCloudEvent() *apicommon.Event {
	return &apicommon.Event{
		Context: apicommon.EventContext{
			CloudEventsVersion: common.CloudEventsVersion,
			EventID:            fmt.Sprintf("%x", "123"),
			ContentType:        "application/json",
			EventTime:          metav1.MicroTime{Time: time.Now().UTC()},
			EventType:          "test",
			EventTypeVersion:   common.CloudEventsVersion,
			Source: &apicommon.URI{
				Host: common.DefaultEventSourceName("test-gateway", "test"),
			},
		},
		Payload: []byte(`{
			"x": "abc"
		}`),
	}
}

func TestEventHandler(t *testing.T) {
	sensor, err := getSensor()
	convey.Convey("Given a sensor spec, create a sensor", t, func() {
		convey.So(err, convey.ShouldBeNil)
		convey.So(sensor, convey.ShouldNotBeNil)
		sec := getsensorExecutionCtx(sensor)

		sec.sensor, err = sec.sensorClient.ArgoprojV1alpha1().Sensors(sensor.Namespace).Create(sensor)
		convey.So(err, convey.ShouldBeNil)

		sec.sensor.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
		fmt.Println(sensor.NodeID("test-gateway:test"))

		sensor2.InitializeNode(sec.sensor, "test-gateway:test", v1alpha1.NodeTypeEventDependency, sec.log, "node is init")
		sensor2.MarkNodePhase(sec.sensor, "test-gateway:test", v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, sec.log, "node is active")

		sensor2.InitializeNode(sec.sensor, "test-workflow-trigger", v1alpha1.NodeTypeTrigger, sec.log, "trigger is init")

		sec.processUpdateNotification(&updateNotification{
			event:            getCloudEvent(),
			notificationType: v1alpha1.EventNotification,
			writer:           &mockHttpWriter{},
			eventDependency: &v1alpha1.EventDependency{
				Name: "test-gateway:test",
			},
		})

		convey.Convey("Update sensor event dependencies", func() {
			sensor = sec.sensor.DeepCopy()
			sensor.Spec.Dependencies = append(sensor.Spec.Dependencies, v1alpha1.EventDependency{
				Name: "test-gateway:test2",
			})
			sec.processUpdateNotification(&updateNotification{
				event:            nil,
				notificationType: v1alpha1.ResourceUpdateNotification,
				writer:           &mockHttpWriter{},
				eventDependency: &v1alpha1.EventDependency{
					Name: "test-gateway:test2",
				},
				sensor: sensor,
			})
			convey.So(len(sec.sensor.Status.Nodes), convey.ShouldEqual, 3)
		})

	})
}

func TestDeleteStaleStatusNodes(t *testing.T) {
	convey.Convey("Given a sensor, delete the stale status nodes", t, func() {
		sensor, err := getSensor()
		convey.So(err, convey.ShouldBeNil)
		sec := getsensorExecutionCtx(sensor)
		nodeId1 := sensor.NodeID("test-gateway:test")
		nodeId2 := sensor.NodeID("test-gateway:test2")
		sec.sensor.Status.Nodes = map[string]v1alpha1.NodeStatus{
			nodeId1: v1alpha1.NodeStatus{
				Type:  v1alpha1.NodeTypeEventDependency,
				Name:  "test-gateway:test",
				Phase: v1alpha1.NodePhaseActive,
				ID:    "1234",
			},
			nodeId2: v1alpha1.NodeStatus{
				Type:  v1alpha1.NodeTypeEventDependency,
				Name:  "test-gateway:test2",
				Phase: v1alpha1.NodePhaseActive,
				ID:    "2345",
			},
		}

		_, ok := sec.sensor.Status.Nodes[nodeId1]
		convey.So(ok, convey.ShouldEqual, true)
		_, ok = sec.sensor.Status.Nodes[nodeId2]
		convey.So(ok, convey.ShouldEqual, true)

		sec.deleteStaleStatusNodes()
		convey.So(len(sec.sensor.Status.Nodes), convey.ShouldEqual, 1)
		_, ok = sec.sensor.Status.Nodes[nodeId1]
		convey.So(ok, convey.ShouldEqual, true)
		_, ok = sec.sensor.Status.Nodes[nodeId2]
		convey.So(ok, convey.ShouldEqual, false)
	})
}

func TestValidateEvent(t *testing.T) {
	convey.Convey("Given an event, validate it", t, func() {
		s, _ := getSensor()
		sec := getsensorExecutionCtx(s)
		dep, valid := sec.validateEvent(&apicommon.Event{
			Context: apicommon.EventContext{
				Source: &apicommon.URI{
					Host: "test-gateway:test",
				},
			},
		})
		convey.So(valid, convey.ShouldEqual, true)
		convey.So(dep, convey.ShouldNotBeNil)
	})
}

func TestParseEvent(t *testing.T) {
	convey.Convey("Given an event payload, parse event", t, func() {
		s, _ := getSensor()
		sec := getsensorExecutionCtx(s)
		e := &apicommon.Event{
			Payload: []byte("hello"),
			Context: apicommon.EventContext{
				Source: &apicommon.URI{
					Host: "test-gateway:test",
				},
			},
		}
		payload, err := json.Marshal(e)
		convey.So(err, convey.ShouldBeNil)

		event, err := sec.parseEvent(payload)
		convey.So(err, convey.ShouldBeNil)
		convey.So(string(event.Payload), convey.ShouldEqual, "hello")
	})
}

func TestSendToInternalQueue(t *testing.T) {
	convey.Convey("Given an event, send it on internal queue", t, func() {
		s, _ := getSensor()
		sec := getsensorExecutionCtx(s)
		e := &apicommon.Event{
			Payload: []byte("hello"),
			Context: apicommon.EventContext{
				Source: &apicommon.URI{
					Host: "test-gateway:test",
				},
			},
		}
		go func() {
			<-sec.queue
		}()
		ok := sec.sendEventToInternalQueue(e, &mockHttpWriter{})
		convey.So(ok, convey.ShouldEqual, true)
	})
}

func TestHandleHttpEventHandler(t *testing.T) {
	convey.Convey("Test http handler", t, func() {
		s, _ := getSensor()
		sec := getsensorExecutionCtx(s)
		e := &apicommon.Event{
			Payload: []byte("hello"),
			Context: apicommon.EventContext{
				Source: &apicommon.URI{
					Host: "test-gateway:test",
				},
			},
		}
		go func() {
			<-sec.queue
		}()
		payload, err := json.Marshal(e)
		convey.So(err, convey.ShouldBeNil)
		writer := &mockHttpWriter{}
		sec.httpEventHandler(writer, &http.Request{
			Body: ioutil.NopCloser(bytes.NewReader(payload)),
		})
		convey.So(writer.Status, convey.ShouldEqual, http.StatusOK)
	})
}

func TestSuccessNatsConnection(t *testing.T) {
	convey.Convey("Given a successful nats connection, generate K8s event", t, func() {
		s, _ := getSensor()
		sec := getsensorExecutionCtx(s)
		sec.successNatsConnection()
		req1, err := labels.NewRequirement(common.LabelOperation, selection.Equals, []string{"nats_connection_setup"})
		convey.So(err, convey.ShouldBeNil)
		req2, err := labels.NewRequirement(common.LabelEventType, selection.Equals, []string{string(common.OperationSuccessEventType)})
		convey.So(err, convey.ShouldBeNil)
		req3, err := labels.NewRequirement(common.LabelSensorName, selection.Equals, []string{string(sec.sensor.Name)})
		convey.So(err, convey.ShouldBeNil)

		eventList, err := sec.kubeClient.CoreV1().Events(sec.sensor.Namespace).List(metav1.ListOptions{
			LabelSelector: labels.NewSelector().Add([]labels.Requirement{*req1, *req2, *req3}...).String(),
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(eventList.Items), convey.ShouldEqual, 1)
		event := eventList.Items[0]
		convey.So(event.Reason, convey.ShouldEqual, "connection setup successfully")
	})
}

func TestEscalateNatsConnectionFailure(t *testing.T) {
	convey.Convey("Given a failed nats connection, escalate through K8s event", t, func() {
		s, _ := getSensor()
		sec := getsensorExecutionCtx(s)
		sec.escalateNatsConnectionFailure()
		req1, err := labels.NewRequirement(common.LabelOperation, selection.Equals, []string{"nats_connection_setup"})
		convey.So(err, convey.ShouldBeNil)
		req2, err := labels.NewRequirement(common.LabelEventType, selection.Equals, []string{string(common.OperationFailureEventType)})
		convey.So(err, convey.ShouldBeNil)
		req3, err := labels.NewRequirement(common.LabelSensorName, selection.Equals, []string{string(sec.sensor.Name)})
		convey.So(err, convey.ShouldBeNil)

		eventList, err := sec.kubeClient.CoreV1().Events(sec.sensor.Namespace).List(metav1.ListOptions{
			LabelSelector: labels.NewSelector().Add([]labels.Requirement{*req1, *req2, *req3}...).String(),
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(eventList.Items), convey.ShouldEqual, 1)
		event := eventList.Items[0]
		convey.So(event.Reason, convey.ShouldEqual, "connection setup failed")
	})
}

func TestSuccessNatsSubscription(t *testing.T) {
	convey.Convey("Given a successful nats subscription, generate K8s event", t, func() {
		s, _ := getSensor()
		eventSource := "fake"
		sec := getsensorExecutionCtx(s)
		sec.successNatsSubscription(eventSource)
		req1, err := labels.NewRequirement(common.LabelOperation, selection.Equals, []string{"nats_subscription_success"})
		convey.So(err, convey.ShouldBeNil)
		req2, err := labels.NewRequirement(common.LabelEventType, selection.Equals, []string{string(common.OperationSuccessEventType)})
		convey.So(err, convey.ShouldBeNil)
		req3, err := labels.NewRequirement(common.LabelSensorName, selection.Equals, []string{string(sec.sensor.Name)})
		convey.So(err, convey.ShouldBeNil)
		req4, err := labels.NewRequirement(common.LabelEventSource, selection.Equals, []string{strings.Replace(eventSource, ":", "_", -1)})
		convey.So(err, convey.ShouldBeNil)

		eventList, err := sec.kubeClient.CoreV1().Events(sec.sensor.Namespace).List(metav1.ListOptions{
			LabelSelector: labels.NewSelector().Add([]labels.Requirement{*req1, *req2, *req3, *req4}...).String(),
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(eventList.Items), convey.ShouldEqual, 1)
		event := eventList.Items[0]
		convey.So(event.Reason, convey.ShouldEqual, "nats subscription success")
	})
}

func TestEscalateNatsSubscriptionFailure(t *testing.T) {
	convey.Convey("Given a failed nats subscription, escalate K8s event", t, func() {
		s, _ := getSensor()
		eventSource := "fake"
		sec := getsensorExecutionCtx(s)
		sec.escalateNatsSubscriptionFailure(eventSource)
		req1, err := labels.NewRequirement(common.LabelOperation, selection.Equals, []string{"nats_subscription_failure"})
		convey.So(err, convey.ShouldBeNil)
		req2, err := labels.NewRequirement(common.LabelEventType, selection.Equals, []string{string(common.OperationFailureEventType)})
		convey.So(err, convey.ShouldBeNil)
		req3, err := labels.NewRequirement(common.LabelSensorName, selection.Equals, []string{string(sec.sensor.Name)})
		convey.So(err, convey.ShouldBeNil)
		req4, err := labels.NewRequirement(common.LabelEventSource, selection.Equals, []string{strings.Replace(eventSource, ":", "_", -1)})
		convey.So(err, convey.ShouldBeNil)

		eventList, err := sec.kubeClient.CoreV1().Events(sec.sensor.Namespace).List(metav1.ListOptions{
			LabelSelector: labels.NewSelector().Add([]labels.Requirement{*req1, *req2, *req3, *req4}...).String(),
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(eventList.Items), convey.ShouldEqual, 1)
		event := eventList.Items[0]
		convey.So(event.Reason, convey.ShouldEqual, "nats subscription failed")
	})
}

func TestProcessNatsMessage(t *testing.T) {
	convey.Convey("Given nats message, process it", t, func() {
		s, _ := getSensor()
		sec := getsensorExecutionCtx(s)
		e := &apicommon.Event{
			Payload: []byte("hello"),
			Context: apicommon.EventContext{
				Source: &apicommon.URI{
					Host: "test-gateway:test",
				},
			},
		}
		dataCh := make(chan []byte)
		go func() {
			data := <-sec.queue
			dataCh <- data.event.Payload
		}()
		payload, err := json.Marshal(e)
		convey.So(err, convey.ShouldBeNil)
		sec.processNatsMessage(payload, "fake")
		data := <-dataCh
		convey.So(data, convey.ShouldNotBeNil)
		convey.So(string(data), convey.ShouldEqual, "hello")
	})
}
