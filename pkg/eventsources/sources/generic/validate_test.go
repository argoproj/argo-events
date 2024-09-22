package generic

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestEventListener_ValidateEventSource(t *testing.T) {
	listener := &EventListener{}

	err := listener.ValidateEventSource(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "server url can't be empty", err.Error())

	content, err := os.ReadFile(fmt.Sprintf("%s/%s", sources.EventSourceDir, "generic.yaml"))
	assert.Nil(t, err)

	var eventSource *v1alpha1.EventSource
	err = yaml.Unmarshal(content, &eventSource)
	assert.Nil(t, err)
	assert.NotNil(t, eventSource.Spec.Generic)

	for name, value := range eventSource.Spec.Generic {
		fmt.Println(name)
		l := &EventListener{
			GenericEventSource: value,
		}
		err := l.ValidateEventSource(context.Background())
		assert.NoError(t, err)
	}
}
