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

//
//var sensorStr = `apiVersion: argoproj.io/v1alpha1
//kind: Sensor
//metadata:
//  name: webhook-sensor
//  labels:
//    sensors.argoproj.io/sensor-controller-instanceid: argo-events
//spec:
//  repeat: true
//  deploySpec:
//    containers:
//    - name: sensor
//      image: "argoproj/sensor"
//      imagePullPolicy: "Always"
//      command: ["/bin/sensor"]
//    serviceAccountName: "argo-events-sa"
//  signals:
//    - name: test-gateway/test-config
//  triggers:
//    - name: webhook-workflow-trigger
//      resource:
//        namespace: argo-events
//        group: argoproj.io
//        version: v1alpha1
//        kind: Workflow
//        source:
//          inline: |
//              apiVersion: argoproj.io/v1alpha1
//              kind: Workflow
//              metadata:
//                generateName: hello-world-
//              spec:
//                entrypoint: whalesay
//                templates:
//                  - name: whalesay
//                    container:
//                      args:
//                        - "hello world"
//                      command:
//                        - cowsay
//                      image: "docker/whalesay:latest"`
//
//func getSensor() (*v1alpha1.Sensor, error) {
//	var sensor v1alpha1.Sensor
//	err := yaml.Unmarshal([]byte(sensorStr), &sensor)
//	return &sensor, err
//}
//
//func getsensorExecutionCtx(sensor *v1alpha1.Sensor) *sensorExecutionCtx {
//	kubeClientset := fake.NewSimpleClientset()
//	return &sensorExecutionCtx{
//		kubeClient:      kubeClientset,
//		discoveryClient: kubeClientset.Discovery().(*discoveryFake.FakeDiscovery),
//		clientPool:      NewFakeClientPool(),
//		log:             zerolog.New(os.Stdout).With().Str("sensor-name", sensor.ObjectMeta.Name).Caller().Logger(),
//		wg:              &sync.WaitGroup{},
//		sensorClient:    sensorFake.NewSimpleClientset(),
//		sensor:          sensor,
//	}
//}
//
//func getCloudEvent() *v1alpha1.Event {
//	return &v1alpha1.Event{
//		Context: v1alpha1.EventContext{
//			CloudEventsVersion: common.CloudEventsVersion,
//			EventID:            fmt.Sprintf("%x", "123"),
//			ContentType:        "application/json",
//			EventTime:          metav1.MicroTime{Time: time.Now().UTC()},
//			EventType:          "test",
//			EventTypeVersion:   common.CloudEventsVersion,
//			Source: &v1alpha1.URI{
//				Host: common.DefaultGatewayConfigurationName("test-gateway", "test-config"),
//			},
//		},
//		Payload: []byte(`{
//			"x": "abc"
//		}`),
//	}
//}
//
//func TestSensorExecutionCtx_signals_and_triggers(t *testing.T) {
//	sensor, err := getSensor()
//	assert.Nil(t, err)
//	assert.NotNil(t, sensor)
//	se := getsensorExecutionCtx(sensor)
//	assert.NotNil(t, se)
//
//	// create the sensor
//	se.sensor, err = se.sensorClient.ArgoprojV1alpha1().Sensors(sensor.Namespace).Create(sensor)
//	assert.Nil(t, err)
//	assert.NotNil(t, se.sensor)
//
//	event := getCloudEvent()
//
//	payload, err := json.Marshal(event)
//	assert.Nil(t, err)
//	assert.NotNil(t, payload)
//
//	fmt.Println(event)
//
//	selectedSignal, valid := se.validateEvent(event)
//	assert.Equal(t, true, valid)
//	assert.NotNil(t, selectedSignal)
//	assert.Equal(t, event.Context.Source.Host, selectedSignal.Name)
//
//	// test persist updates
//	se.sensor.Status.Phase = v1alpha1.NodePhaseError
//	err = se.persistUpdates()
//	assert.Nil(t, err)
//	assert.Equal(t, v1alpha1.NodePhaseError, se.sensor.Status.Phase)
//
//	updateNotification := &v1alpha1.EventWrapper{
//		Seen:  true,
//		Event: *event,
//	}
//	eventWrapperBytes, err := yaml.Marshal(updateNotification)
//	assert.Nil(t, err)
//	assert.NotNil(t, eventWrapperBytes)
//
//	testNode := v1alpha1.NodeStatus{
//		Phase: v1alpha1.NodePhaseNew,
//		Name:  common.DefaultGatewayConfigurationName("test-gateway", "test-config"),
//		StartedAt: metav1.MicroTime{
//			Time: time.Now(),
//		},
//		Type:        v1alpha1.NodeTypeEventDependency,
//		DisplayName: common.DefaultGatewayConfigurationName("test-gateway", "test-config"),
//	}
//	se.sensor.Status.Nodes = map[string]v1alpha1.NodeStatus{
//		sensor.NodeID(common.DefaultGatewayConfigurationName("test-gateway", "test-config")): testNode,
//	}
//
//	for _, node := range se.sensor.Status.Nodes {
//		if node.Type == v1alpha1.NodeTypeEventDependency {
//			se.markNodePhase(node.Name, v1alpha1.NodePhaseComplete, "node is completed")
//		}
//	}
//
//	err = se.processTriggers()
//	assert.Nil(t, err)
//
//	for _, node := range se.sensor.Status.Nodes {
//		if node.Type == v1alpha1.NodeTypeTrigger {
//			assert.Equal(t, v1alpha1.NodePhaseComplete, node.Phase)
//		}
//	}
//
//	// test markNodePhase
//	nodeStatus := se.markNodePhase(common.DefaultGatewayConfigurationName("test-gateway", "test-config"), v1alpha1.NodePhaseError, "error phase")
//	assert.Nil(t, err)
//	assert.NotNil(t, nodeStatus)
//	assert.Equal(t, string(v1alpha1.NodePhaseError), string(nodeStatus.Phase))
//
//	// persist updates
//	err = se.persistUpdates()
//	assert.Nil(t, err)
//}
