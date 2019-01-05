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

//
//func Test_markGatewayNodePhase(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	status := gc.markGatewayNodePhase("1234", v1alpha1.NodePhaseInitialized, "init")
//	assert.Nil(t, status)
//
//	gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
//
//	gc.gw.Status.Nodes["1234"] = v1alpha1.NodeStatus{
//		ID:     "1234",
//		TimeID: "5678",
//		Name:   "test",
//	}
//
//	status = gc.markGatewayNodePhase("1234", v1alpha1.NodePhaseInitialized, "init")
//	assert.NotNil(t, status)
//	assert.Equal(t, v1alpha1.NodePhaseInitialized, status.Phase)
//}
//
//func Test_getNodeByID(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	status := gc.getNodeByID("test")
//	assert.Nil(t, status)
//	gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
//
//	gc.gw.Status.Nodes["test"] = v1alpha1.NodeStatus{
//		ID:     "1234",
//		TimeID: "5678",
//		Name:   "test",
//	}
//	status = gc.getNodeByID("test")
//	assert.NotNil(t, status)
//}
//
//func Test_initializeNode(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	status := gc.initializeNode("test", "test-node", "1234", "init")
//	assert.NotNil(t, status)
//	assert.Equal(t, string(v1alpha1.NodePhaseInitialized), string(status.Phase))
//	status1 := gc.initializeNode("test", "test-node", "1234", "init")
//	assert.NotNil(t, status1)
//	assert.Equal(t, status, status1)
//}
//
//func Test_persistUpdates(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	gc.gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Create(gw)
//	assert.Nil(t, err)
//	gc.gw.Spec.Type = "change-type"
//	err = gc.persistUpdates()
//	assert.Nil(t, err)
//	_, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Get(gc.gw.Name, metav1.GetOptions{})
//	assert.Equal(t, "change-type", gc.gw.Spec.Type)
//}
//
//func Test_reapplyupdate(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	gc.gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Create(gw)
//	assert.Nil(t, err)
//	gc.gw.Spec.Type = "change-type"
//	err = gc.reapplyUpdate()
//	assert.Nil(t, err)
//}
//
//func Test_updateGatewayResourceNoGateway(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	e := gc.EventSourceStateChangeK8sEvent("test", "test", &EventSourceData{
//		Config: "testConfig",
//		Src:    "testSrc",
//		ID:     "1234",
//		TimeID: "4567",
//	})
//	e, err = common.CreateK8Event(e, gc.Clientset)
//	assert.Nil(t, err)
//
//	err = gc.UpdateGatewayEventSourceState(e)
//	assert.NotNil(t, err)
//
//	e, err = gc.Clientset.CoreV1().Events(gc.gw.Namespace).Get(e.Name, metav1.GetOptions{})
//	assert.Nil(t, err)
//	assert.Equal(t, "true", e.ObjectMeta.Labels[common.LabelEventSeen])
//}
//
//func Test_updateGatewayResourceInit(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	e := gc.EventSourceStateChangeK8sEvent("test", v1alpha1.NodePhaseInitialized, &EventSourceData{
//		Config: "testConfig",
//		Src:    "testSrc",
//		ID:     "1234",
//		TimeID: "4567",
//	})
//	e, err = common.CreateK8Event(e, gc.Clientset)
//	assert.Nil(t, err)
//
//	_, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Create(gw)
//	assert.Nil(t, err)
//
//	err = gc.UpdateGatewayEventSourceState(e)
//	assert.Nil(t, err)
//
//	gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Get(gc.gw.Name, metav1.GetOptions{})
//	assert.Nil(t, err)
//	assert.NotNil(t, gc.gw.Status.Nodes)
//	for _, node := range gc.gw.Status.Nodes {
//		assert.Equal(t, string(v1alpha1.NodePhaseInitialized), string(node.Phase))
//	}
//
//	e, err = gc.Clientset.CoreV1().Events(gc.gw.Namespace).Get(e.Name, metav1.GetOptions{})
//	assert.Nil(t, err)
//	assert.Equal(t, "true", e.ObjectMeta.Labels[common.LabelEventSeen])
//}
//
//func Test_updateGatewayResourceRunning(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	e := gc.EventSourceStateChangeK8sEvent("test", v1alpha1.NodePhaseRunning, &EventSourceData{
//		Config: "testConfig",
//		Src:    "testSrc",
//		ID:     "1234",
//		TimeID: "4567",
//	})
//	e, err = common.CreateK8Event(e, gc.Clientset)
//	assert.Nil(t, err)
//
//	gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
//	gc.gw.Status.Nodes[e.Labels[common.LabelGatewayEventSourceID]] = v1alpha1.NodeStatus{
//		Name:   "test-node",
//		Phase:  v1alpha1.NodePhaseInitialized,
//		ID:     "1234",
//		TimeID: "4567",
//	}
//
//	_, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Create(gc.gw)
//	assert.Nil(t, err)
//
//	err = gc.UpdateGatewayEventSourceState(e)
//	assert.Nil(t, err)
//
//	gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Get(gc.gw.Name, metav1.GetOptions{})
//	assert.Nil(t, err)
//	for _, node := range gc.gw.Status.Nodes {
//		assert.Equal(t, string(v1alpha1.NodePhaseRunning), string(node.Phase))
//	}
//
//	e, err = gc.Clientset.CoreV1().Events(gc.gw.Namespace).Get(e.Name, metav1.GetOptions{})
//	assert.Nil(t, err)
//	assert.Equal(t, "true", e.ObjectMeta.Labels[common.LabelEventSeen])
//}
//
//func Test_updateGatewayResourceRemove(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	e := gc.EventSourceStateChangeK8sEvent("test", v1alpha1.NodePhaseRemove, &EventSourceData{
//		Config: "testConfig",
//		Src:    "testSrc",
//		ID:     "1234",
//		TimeID: "4567",
//	})
//	e, err = common.CreateK8Event(e, gc.Clientset)
//	assert.Nil(t, err)
//
//	gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
//	gc.gw.Status.Nodes[e.Labels[common.LabelGatewayEventSourceID]] = v1alpha1.NodeStatus{
//		Name:   "test-node",
//		Phase:  v1alpha1.NodePhaseRunning,
//		ID:     "1234",
//		TimeID: "4567",
//	}
//
//	_, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Create(gc.gw)
//	assert.Nil(t, err)
//
//	err = gc.UpdateGatewayEventSourceState(e)
//	assert.Nil(t, err)
//	assert.Equal(t, 0, len(gc.gw.Status.Nodes))
//
//	e, err = gc.Clientset.CoreV1().Events(gc.gw.Namespace).Get(e.Name, metav1.GetOptions{})
//	assert.Nil(t, err)
//	assert.Equal(t, "true", e.ObjectMeta.Labels[common.LabelEventSeen])
//}
//
//func Test_updateGatewayResourceCompleted(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gc := newGatewayconfig(gw)
//	e := gc.EventSourceStateChangeK8sEvent("test", v1alpha1.NodePhaseCompleted, &EventSourceData{
//		Config: "testConfig",
//		Src:    "testSrc",
//		ID:     "1234",
//		TimeID: "4567",
//	})
//	e, err = common.CreateK8Event(e, gc.Clientset)
//	assert.Nil(t, err)
//
//	gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
//	gc.gw.Status.Nodes[e.Labels[common.LabelGatewayEventSourceID]] = v1alpha1.NodeStatus{
//		Name:   "test-node",
//		Phase:  v1alpha1.NodePhaseRunning,
//		ID:     "1234",
//		TimeID: "4567",
//	}
//
//	_, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Create(gc.gw)
//	assert.Nil(t, err)
//
//	err = gc.UpdateGatewayEventSourceState(e)
//	assert.Nil(t, err)
//	for _, node := range gc.gw.Status.Nodes {
//		assert.Equal(t, string(v1alpha1.NodePhaseCompleted), string(node.Phase))
//	}
//
//	e, err = gc.Clientset.CoreV1().Events(gc.gw.Namespace).Get(e.Name, metav1.GetOptions{})
//	assert.Nil(t, err)
//	assert.Equal(t, "true", e.ObjectMeta.Labels[common.LabelEventSeen])
//}
