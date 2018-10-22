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

package common

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

// TestCreateK8Event tests creation of k8 event
func TestCreateK8Event(t *testing.T) {
	event := &K8Event{
		Name:                "test-event",
		Action:              "test action",
		Type:                "test",
		Labels:              map[string]string{},
		Namespace:           "default",
		Kind:                "test-component",
		Reason:              "for testing purposes",
		ReportingController: "test-controller",
		ReportingInstance:   "1",
	}

	k8Event := GetK8Event(event)
	generatedEvent, err := CreateK8Event(k8Event, fake.NewSimpleClientset())
	assert.Equal(t, err, nil)
	assert.Equal(t, generatedEvent.Name, k8Event.Name)
}
