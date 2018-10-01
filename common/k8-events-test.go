package common

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

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
