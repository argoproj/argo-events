/*
Copyright 2018 The Argoproj Authors.

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

package gitlab

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
)

func TestValidateEventSource(t *testing.T) {
	listener := &EventListener{}

	err := listener.ValidateEventSource(context.Background())
	assert.Error(t, err)

	content, err := os.ReadFile(fmt.Sprintf("%s/%s", sources.EventSourceDir, "gitlab.yaml"))
	assert.Nil(t, err)

	var eventSource *v1alpha1.EventSource
	err = yaml.Unmarshal(content, &eventSource)
	assert.Nil(t, err)
	assert.NotNil(t, eventSource.Spec.Gitlab)

	for name, value := range eventSource.Spec.Gitlab {
		fmt.Println(name)
		l := &EventListener{
			GitlabEventSource: value,
		}
		err := l.ValidateEventSource(context.Background())
		assert.NoError(t, err)
	}
}
