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
	"testing"

	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTransformEvent(t *testing.T) {
	event := &gateways.Event{
		Name:    "hello",
		Payload: []byte("{\"name\": \"hello\"}"),
	}
	ctx := &GatewayContext{
		gateway: &v1alpha1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-gateway",
			},
			Spec: v1alpha1.GatewaySpec{
				Type: "webhook",
				EventSourceRef: &v1alpha1.EventSourceRef{
					Name: "test-event-source",
				},
			},
		},
	}
	cloudevent, err := ctx.transformEvent(event)
	assert.Nil(t, err)
	assert.NotNil(t, cloudevent.Context.AsV03())
	assert.Equal(t, "test-event-source", cloudevent.Source())
	assert.Equal(t, "hello", cloudevent.Subject())
	assert.Equal(t, "webhook", cloudevent.Type())

	data, err := cloudevent.DataBytes()
	assert.Nil(t, err)
	assert.Equal(t, string(data), "{\"name\": \"hello\"}")
}
