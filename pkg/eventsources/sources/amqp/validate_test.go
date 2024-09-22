package amqp

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
	assert.Equal(t, "either url or urlSecret must be specified", err.Error())

	content, err := os.ReadFile(fmt.Sprintf("%s/%s", sources.EventSourceDir, "amqp.yaml"))
	assert.Nil(t, err)

	var eventSource *v1alpha1.EventSource
	err = yaml.Unmarshal(content, &eventSource)
	assert.Nil(t, err)
	assert.NotNil(t, eventSource.Spec.AMQP)

	for _, value := range eventSource.Spec.AMQP {
		l := &EventListener{
			AMQPEventSource: value,
		}
		err := l.ValidateEventSource(context.Background())
		assert.NoError(t, err)
	}
}
