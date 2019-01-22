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
	"fmt"
	"github.com/argoproj/argo-events/common"
	sensor2 "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"testing"

	sensorFake "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoveryFake "k8s.io/client-go/discovery/fake"
	"time"
)

var sensorStr = `apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: test-sensor
  labels:
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  deploySpec:
    containers:
      - name: "sensor"
        image: "metalgearsolid/sensor:v0.6.1"
        imagePullPolicy: Always
    serviceAccountName: argo-events-sa
  dependencies:
    - name: test-gateway:test
  triggers:
    - name: test-workflow-trigger
      resource:
        namespace: argo-events
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
                  -
                    container:
                      args:
                        - "hello world"
                      command:
                        - cowsay
                      image: "docker/whalesay:latest"
                    name: whalesay`

func getSensor() (*v1alpha1.Sensor, error) {
	var sensor v1alpha1.Sensor
	err := yaml.Unmarshal([]byte(sensorStr), &sensor)
	return &sensor, err
}

type mockHttpWriter struct{}

func (m *mockHttpWriter) Header() http.Header {
	return http.Header{}
}

func (m *mockHttpWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockHttpWriter) WriteHeader(statusCode int) {

}

func getsensorExecutionCtx(sensor *v1alpha1.Sensor) *sensorExecutionCtx {
	kubeClientset := fake.NewSimpleClientset()
	return &sensorExecutionCtx{
		kubeClient:           kubeClientset,
		discoveryClient:      kubeClientset.Discovery().(*discoveryFake.FakeDiscovery),
		clientPool:           NewFakeClientPool(),
		log:                  common.GetLoggerContext(common.LoggerConf()).Logger(),
		sensorClient:         sensorFake.NewSimpleClientset(),
		sensor:               sensor,
		controllerInstanceID: "test-1",
	}
}

func getCloudEvent() *v1alpha1.Event {
	return &v1alpha1.Event{
		Context: v1alpha1.EventContext{
			CloudEventsVersion: common.CloudEventsVersion,
			EventID:            fmt.Sprintf("%x", "123"),
			ContentType:        "application/json",
			EventTime:          metav1.MicroTime{Time: time.Now().UTC()},
			EventType:          "test",
			EventTypeVersion:   common.CloudEventsVersion,
			Source: &v1alpha1.URI{
				Host: common.DefaultGatewayConfigurationName("test-gateway", "test"),
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

		sensor2.InitializeNode(sec.sensor, "test-gateway:test", v1alpha1.NodeTypeEventDependency, &sec.log, "node is init")
		sensor2.MarkNodePhase(sec.sensor, "test-gateway:test", v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, &sec.log, "node is active")

		sensor2.InitializeNode(sec.sensor, "test-workflow-trigger", v1alpha1.NodeTypeTrigger, &sec.log, "trigger is init")

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
