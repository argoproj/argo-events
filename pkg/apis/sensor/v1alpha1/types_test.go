package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvent_DataString(t *testing.T) {
	assert.Empty(t, Event{}.DataString())
	assert.Equal(t, "bXktZGF0YQ==", Event{Data: []byte("my-data")}.DataString())
	assert.Equal(t, `"my-data"`, Event{Data: []byte(`"my-data"`), Context: &EventContext{DataContentType: "application/json"}}.DataString())
}
