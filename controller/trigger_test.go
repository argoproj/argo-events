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
	"strconv"
	"testing"

	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/gnatsd/test"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"

	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
)

var sampleTrigger = v1alpha1.Trigger{
	Name: "sample",
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
}

func TestProcessTrigger(t *testing.T) {
	fake := newFakeController()
	defer fake.teardown()

	triggers := make([]v1alpha1.Trigger, 1)
	triggers[0] = sampleTrigger
	sampleSensor.Spec.Triggers = triggers

	soc := newSensorOperationCtx(&sampleSensor, fake.SensorController)

	node, err := soc.processTrigger(sampleTrigger)
	assert.NotNil(t, err)
	// assert node was processed correctly
	assert.Equal(t, sampleTrigger.Name, node.Name)
	assert.Equal(t, sampleTrigger.Name, node.DisplayName)
	assert.Equal(t, v1alpha1.NodePhaseError, node.Phase)
	assert.Equal(t, v1alpha1.NodeTypeTrigger, node.Type)
	// assert node equality with operationContext
	assert.Equal(t, *node, soc.s.Status.Nodes[node.ID])

	// now force node status to resolved
	node.Phase = v1alpha1.NodePhaseResolved
	soc.s.Status.Nodes[node.ID] = *node
	node, err = soc.processTrigger(sampleTrigger)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha1.NodePhaseSucceeded, node.Phase)
}

func TestSendMessage(t *testing.T) {
	natsEmbeddedServerOpts := server.Options{
		Host:           "localhost",
		Port:           4224,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 256,
	}
	testServer := test.RunServer(&natsEmbeddedServerOpts)
	defer testServer.Shutdown()

	unsupportedMsg := &v1alpha1.Message{
		Body:   "",
		Stream: v1alpha1.Stream{},
	}
	err := sendMessage(unsupportedMsg)
	assert.NotNil(t, err)

	supportedMsg := &v1alpha1.Message{
		Body: "",
		Stream: v1alpha1.Stream{
			Type:       "NATS",
			URL:        "nats://" + natsEmbeddedServerOpts.Host + ":" + strconv.Itoa(natsEmbeddedServerOpts.Port),
			Attributes: map[string]string{"subject": "test"},
		},
	}
	err = sendMessage(supportedMsg)
	assert.Nil(t, err)
}
