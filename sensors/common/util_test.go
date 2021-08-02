package common

import (
	"fmt"
	"strings"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func createFakeEvent(eventID string) *v1alpha1.Event {
	return &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			ID: eventID,
		},
	}
}

func TestApplyEventLabels(t *testing.T) {
	const maxK8sLabelLen int = 63
	const labelNamePrefix string = "events.argoproj.io/event-"

	t.Run("test event label name and value of a single event", func(t *testing.T) {
		uid := uuid.New()
		labelsMap := make(map[string]string)
		fakeEventsMap := map[string]*v1alpha1.Event{
			"test-dep": createFakeEvent(fmt.Sprintf("%x", uid)),
		}

		labelName := fmt.Sprintf("%s0", labelNamePrefix)
		err := ApplyEventLabels(labelsMap, fakeEventsMap)
		assert.Nil(t, err)
		assert.Equal(t, uid.String(), labelsMap[labelName])
		assert.True(t, len(labelsMap[labelName]) <= maxK8sLabelLen)
	})

	t.Run("test failure in case event ID isn't valid hex string", func(t *testing.T) {
		labelsMap := make(map[string]string)
		fakeEventsMap := map[string]*v1alpha1.Event{
			"test-dep": createFakeEvent("this is not hex"),
		}

		err := ApplyEventLabels(labelsMap, fakeEventsMap)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "failed to decode event ID"))
	})

	t.Run("test event labels prefix and value of multiple events", func(t *testing.T) {
		labelsMap := make(map[string]string)
		fakeEventsMap := map[string]*v1alpha1.Event{
			"test-dep1": createFakeEvent(fmt.Sprintf("%x", uuid.New())),
			"test-dep2": createFakeEvent(fmt.Sprintf("%x", uuid.New())),
		}

		err := ApplyEventLabels(labelsMap, fakeEventsMap)
		assert.Nil(t, err)
		assert.Equal(t, len(fakeEventsMap), len(labelsMap))

		for key, val := range labelsMap {
			assert.True(t, strings.HasPrefix(key, labelNamePrefix))
			_, err := uuid.Parse(val)
			assert.Nil(t, err)
			assert.True(t, len(val) <= maxK8sLabelLen)
		}
	})
}