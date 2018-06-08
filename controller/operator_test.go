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

package controller

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/gnatsd/test"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/blackrock/axis/common"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
)

var sampleSensor = v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "sample",
		Namespace: apiv1.NamespaceDefault,
	},
	Spec: v1alpha1.SensorSpec{
		Signals: []v1alpha1.Signal{
			{
				Name: "nats-test",
				NATS: &v1alpha1.NATS{
					URL:     "nats://sample-test:4222",
					Subject: "testing-in",
				},
			},
			{
				Name: "resource-test",
				Resource: &v1alpha1.ResourceSignal{
					GroupVersionKind: v1alpha1.GroupVersionKind{
						Group:   "argoproj.io",
						Version: "v1alpha1",
						Kind:    "workflow",
					},
					Namespace: apiv1.NamespaceDefault,
				},
			},
		},
		Triggers: []v1alpha1.Trigger{
			{
				Name: "test-trigger",
				Message: &v1alpha1.Message{
					Body: "this is where the message body goes",
					Stream: v1alpha1.Stream{
						NATS: &v1alpha1.NATS{
							URL:     "nats://sample-test:4222",
							Subject: "testing-out",
						},
					},
				},
				Resource: &v1alpha1.ResourceObject{
					Namespace: apiv1.NamespaceDefault,
					GroupVersionKind: v1alpha1.GroupVersionKind{
						Group:   "argoproj.io",
						Version: "v1alpha1",
						Kind:    "workflow",
					},
					ArtifactLocation: &v1alpha1.ArtifactLocation{
						S3: &v1alpha1.S3Artifact{},
					},
					Labels: map[string]string{"test-label": "test-value"},
				},
			},
		},
	},
}

func TestSensorOperateLifecycle(t *testing.T) {
	fake := newFakeController()
	defer fake.teardown()

	sensor, err := fake.sensorClientset.ArgoprojV1alpha1().Sensors(fake.Config.Namespace).Create(&sampleSensor)
	assert.Nil(t, err)
	soc := newSensorOperationCtx(sensor, fake.SensorController)

	// STEP 1: operate on new sensor
	err = soc.operate()
	assert.Nil(t, err)
	// assert the status of sensor's signal is initializing
	assert.Equal(t, v1alpha1.NodePhaseInit, soc.s.Status.Phase)
	assert.Equal(t, 2, len(soc.s.Status.Nodes))
	natsSignalNode := soc.getNodeByName(sampleSensor.Spec.Signals[0].Name)
	assert.Equal(t, v1alpha1.NodePhaseInit, natsSignalNode.Phase)
	resourceSignalNode := soc.getNodeByName(sampleSensor.Spec.Signals[1].Name)
	assert.Equal(t, v1alpha1.NodePhaseInit, resourceSignalNode.Phase)
	fmt.Print("\n\n")

	// STEP 2: operate on still init sensor after creating executor pod
	executorPod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      soc.s.Name + "-sensor-123",
			Namespace: fake.Config.Namespace,
			Labels:    map[string]string{common.LabelJobName: soc.s.Name + "-sensor", common.LabelKeyResolved: "false"},
		},
		Spec: apiv1.PodSpec{},
		Status: apiv1.PodStatus{
			Phase: apiv1.PodRunning,
		},
	}
	_, err = fake.kubeClientset.CoreV1().Pods(fake.Config.Namespace).Create(executorPod)
	assert.Nil(t, err)
	err = soc.operate()
	assert.Nil(t, err)
	// check status of the sensor's signals
	natsSignalNode = soc.getNodeByName(sampleSensor.Spec.Signals[0].Name)
	assert.Equal(t, v1alpha1.NodePhaseActive, natsSignalNode.Phase)
	resourceSignalNode = soc.getNodeByName(sampleSensor.Spec.Signals[1].Name)
	assert.Equal(t, v1alpha1.NodePhaseActive, resourceSignalNode.Phase)
	fmt.Print("\n\n")

	// STEP 3: operate on sensor with resolved signals and succeeded pod
	executorPod.Status.Phase = apiv1.PodSucceeded
	_, err = fake.kubeClientset.CoreV1().Pods(fake.Config.Namespace).Update(executorPod)
	assert.Nil(t, err)
	natsSignalNode.Phase = v1alpha1.NodePhaseResolved
	natsSignalNode.ResolvedAt = metav1.Time{Time: time.Now().UTC()}
	soc.s.Status.Nodes[natsSignalNode.ID] = *natsSignalNode
	resourceSignalNode.Phase = v1alpha1.NodePhaseResolved
	resourceSignalNode.ResolvedAt = metav1.Time{Time: time.Now().UTC()}
	soc.s.Status.Nodes[resourceSignalNode.ID] = *resourceSignalNode
	soc.operate()
	// check status of the sensor's signals
	natsSignalNode = soc.getNodeByName(sampleSensor.Spec.Signals[0].Name)
	assert.Equal(t, v1alpha1.NodePhaseSucceeded, natsSignalNode.Phase)
	resourceSignalNode = soc.getNodeByName(sampleSensor.Spec.Signals[1].Name)
	assert.Equal(t, v1alpha1.NodePhaseSucceeded, resourceSignalNode.Phase)
	// check status of the sensor's trigger
	triggerNode := soc.getNodeByName(soc.s.Spec.Triggers[0].Name)
	assert.Equal(t, v1alpha1.NodePhaseError, triggerNode.Phase)
	assert.Equal(t, v1alpha1.NodePhaseError, soc.s.Status.Phase)

	// STEP 4: operate on sensor with resolved triggers
	triggerNode.Phase = v1alpha1.NodePhaseSucceeded
	soc.s.Status.Nodes[triggerNode.ID] = *triggerNode
	soc.s.Status.Phase = v1alpha1.NodePhaseResolved
	soc.operate()
	//check status of sensor
	assert.Equal(t, v1alpha1.NodePhaseSucceeded, soc.s.Status.Phase)

	// STEP 5: operate on succeeded sensor
	err = soc.operate()
	assert.Nil(t, err)
}

func TestReRunSensor(t *testing.T) {
	fake := newFakeController()
	defer fake.teardown()

	sampleSensor.Status = v1alpha1.SensorStatus{
		Phase: v1alpha1.NodePhaseSucceeded,
		Nodes: map[string]v1alpha1.NodeStatus{
			"testEntry": v1alpha1.NodeStatus{
				ID:    "id",
				Name:  "name",
				Phase: v1alpha1.NodePhaseSucceeded,
			},
		},
	}
	sensor, err := fake.sensorClientset.ArgoprojV1alpha1().Sensors(fake.Config.Namespace).Create(&sampleSensor)
	assert.Nil(t, err)
	soc := newSensorOperationCtx(sensor, fake.SensorController)

	soc.reRunSensor()

	// verify sensor status fields
	assert.True(t, soc.updated)
	assert.True(t, soc.needJobCreation)
	assert.Equal(t, v1alpha1.NodePhaseInit, soc.s.Status.Phase)
	assert.Equal(t, 2, len(soc.s.Status.Nodes))
	natsSignalNode := soc.getNodeByName(sampleSensor.Spec.Signals[0].Name)
	assert.Equal(t, v1alpha1.NodePhaseInit, natsSignalNode.Phase)
	resourceSignalNode := soc.getNodeByName(sampleSensor.Spec.Signals[1].Name)
	assert.Equal(t, v1alpha1.NodePhaseInit, resourceSignalNode.Phase)
}

func TestEscalationSent(t *testing.T) {
	fake := newFakeController()
	defer fake.teardown()

	natsEmbeddedServerOpts := server.Options{
		Host:           "localhost",
		Port:           4225,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 256,
	}
	testServer := test.RunServer(&natsEmbeddedServerOpts)
	defer testServer.Shutdown()

	sampleSensor.Spec.Escalation = v1alpha1.EscalationPolicy{
		Level: "High",
		Message: v1alpha1.Message{
			Body: "esclating this sensor on failure",
			Stream: v1alpha1.Stream{
				NATS: &v1alpha1.NATS{
					URL:     "nats://" + natsEmbeddedServerOpts.Host + ":" + strconv.Itoa(natsEmbeddedServerOpts.Port),
					Subject: "escalation",
				},
			},
		},
	}

	sampleSensor.Status = v1alpha1.SensorStatus{
		Escalated: false,
		Phase:     v1alpha1.NodePhaseError,
		Nodes:     make(map[string]v1alpha1.NodeStatus),
	}
	for _, signal := range sampleSensor.Spec.Signals {
		nodeID := sampleSensor.NodeID(signal.Name)
		sampleSensor.Status.Nodes[nodeID] = v1alpha1.NodeStatus{
			ID:          nodeID,
			Name:        signal.Name,
			DisplayName: signal.Name,
			Type:        v1alpha1.NodeTypeSignal,
			Phase:       v1alpha1.NodePhaseError,
			StartedAt:   metav1.Time{},
			Message:     "failed node reason",
		}
	}

	sensor, err := fake.sensorClientset.ArgoprojV1alpha1().Sensors(fake.Config.Namespace).Create(&sampleSensor)
	assert.Nil(t, err)
	soc := newSensorOperationCtx(sensor, fake.SensorController)

	// create the executor job
	executorJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      soc.s.Name + "-sensor",
			Namespace: fake.Config.Namespace,
			Labels:    map[string]string{common.LabelKeyResolved: "false"},
		},
		Spec: batchv1.JobSpec{},
	}
	_, err = fake.kubeClientset.BatchV1().Jobs(fake.Config.Namespace).Create(executorJob)
	assert.Nil(t, err)

	// create the executor pod
	executorPod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      soc.s.Name + "-sensor-123",
			Namespace: fake.Config.Namespace,
			Labels:    map[string]string{common.LabelJobName: soc.s.Name + "-sensor", common.LabelKeyResolved: "false"},
		},
		Spec: apiv1.PodSpec{},
		Status: apiv1.PodStatus{
			Phase: apiv1.PodFailed,
		},
	}
	_, err = fake.kubeClientset.CoreV1().Pods(fake.Config.Namespace).Create(executorPod)
	assert.Nil(t, err)

	// if sensor not yet escalated, make sure we escalate and update it
	soc.operate()
	assert.True(t, soc.s.Status.Escalated)
	assert.True(t, soc.updated)

	// second pass through, verify we don't escalate and update
	soc.updated = false // reset this field
	soc.operate()
	assert.True(t, soc.s.Status.Escalated)
	assert.False(t, soc.updated)
}

func TestInferFailedPodCause(t *testing.T) {
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "testing",
		},
		Status: apiv1.PodStatus{
			Message: "status message failure here",
		},
	}
	// pod message
	failureCause := inferFailedPodCause(pod)
	assert.Equal(t, "status message failure here", failureCause)

	// annotations message
	annotations := make(map[string]string)
	annotations[common.AnnotationKeyNodeMessage] = "annotation message failure here"
	pod.ObjectMeta.Annotations = annotations
	pod.Status.Message = ""

	failureCause = inferFailedPodCause(pod)
	assert.Equal(t, "annotation message failure here", failureCause)

	// pod status conditions
	delete(pod.ObjectMeta.Annotations, common.AnnotationKeyNodeMessage)
	conditions := make([]apiv1.PodCondition, 1)
	conditions[0] = apiv1.PodCondition{
		Message: "pod status condition message failure",
	}
	pod.Status.Conditions = conditions

	failureCause = inferFailedPodCause(pod)
	assert.Equal(t, "Pod Conditions: pod status condition message failure. ", failureCause)
}
