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

package nats

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/eventsources/sources"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestValidateEventSource(t *testing.T) {
	listener := &EventListener{}

	err := listener.ValidateEventSource(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "url must be specified", err.Error())

	content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", sources.EventSourceDir, "nats.yaml"))
	assert.Nil(t, err)

	var eventSource *v1alpha1.EventSource
	err = yaml.Unmarshal(content, &eventSource)
	assert.Nil(t, err)
	assert.NotNil(t, eventSource.Spec.NATS)

	for name, value := range eventSource.Spec.NATS {
		fmt.Println(name)
		l := &EventListener{
			NATSEventSource: value,
		}
		err := l.ValidateEventSource(context.Background())
		assert.NoError(t, err)
	}
}

func TestValidateEventSourceMissingAuthSecret(t *testing.T) {
	listener := &EventListener{}

	err := listener.ValidateEventSource(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "url must be specified", err.Error())

	content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", sources.EventSourceDir, "nats.yaml"))
	assert.Nil(t, err)

	var eventSource *v1alpha1.EventSource
	err = yaml.Unmarshal(content, &eventSource)
	assert.Nil(t, err)
	assert.NotNil(t, eventSource.Spec.NATS)

	for _, value := range eventSource.Spec.NATS {
		value.Auth = v1alpha1.NATSAuthToken
		l := &EventListener{
			NATSEventSource: value,
		}
		err := l.ValidateEventSource(context.Background())
		assert.Equal(t, err.Error(), "Token secret must be specified")
	}
}
